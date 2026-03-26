# Prediction Model Research

## Why This Document Exists

Vektor already has a working predictor: a bounded first-order Markov chain in the C++ engine. The question is not whether there are more sophisticated models in the world. There are. The real question is which open-source models are plausible upgrades for this specific system without breaking the reason the system exists in the first place: cheap, fast, predictable prefetch decisions on the request path.

That matters because "better model" is easy to say and expensive to mean. In this codebase, every extra allocation, branch, cache miss, serialization hop, and foreign-runtime dependency has to earn its place.

## Operational Constraints

Before comparing models, it helps to name the constraints explicitly.

The current design has a few properties that are easy to underestimate:

- prediction is local and incremental,
- updates happen online as new keys arrive,
- memory is bounded by `max_keys`,
- inference lives next to the hot path in native code,
- prediction output is naturally probabilistic and easy to threshold,
- failure modes are understandable because the state is basically a transition table.

Any successor model should be judged against those properties, not against a leaderboard in isolation.

For Vektor, the relevant optimization targets are:

- per-event prediction latency,
- update cost per observation,
- memory growth under large key cardinality,
- cold-start behavior,
- quality of top-k next-key predictions,
- operational simplicity under continuous traffic.

This is why generic time-series forecasting models are not a fit. Vektor is not forecasting a scalar. It is predicting the next member of a sparse, high-cardinality sequence under tight runtime constraints.

## What The Current Markov Model Gets Right

The existing model is simple, but it is simple in the useful way.

Given a recent history `h = [k_{t-n+1}, ..., k_t]`, it estimates:

`P(k_{t+1} | h)`

In the current implementation, the effective state is the joined history suffix and the score is the normalized transition count for the next key. That gives Vektor a few important advantages:

- updates are `O(1)` on average with hash-table inserts,
- inference is proportional to the outgoing transitions for the matched state,
- the resulting probabilities are directly interpretable,
- the model adapts online without a separate training phase.

The weakness is also clear: a first-order or low-order Markov model forgets structure that lives beyond the local suffix. If real traces contain repeated motifs, branching subroutines, tenant-specific modes, or long-range dependencies, a plain transition table leaves hit rate on the table.

So the goal is not to abandon the stochastic framing. The goal is to preserve the good parts while buying back more sequence context where it actually helps.

## Evaluation Lens

The most honest way to compare models in Vektor is to separate them into three buckets:

1. direct hot-path candidates,
2. sidecar or offline teachers,
3. re-rankers that sit on top of cheap candidate generators.

A model can be very strong in one bucket and still be the wrong choice for another. That distinction is more useful than arguing about which family is "best" in the abstract.

## Model Families Worth Considering

### 1. Variable-order sequence predictors: `CPT+`, `TDAG`, `DG`

Open-source reference:

- [SPMF](https://www.philippe-fournier-viger.com/spmf/)
- [SPMF sequence prediction comparison](https://www.philippe-fournier-viger.com/spmf/CompareSequencePredictionModels.php)

This is the closest conceptual upgrade to the current engine.

These models still treat the problem as sequence prediction rather than representation learning. That matters. They are designed to answer the same basic question as the current Markov chain, but with more expressive use of prior subsequences. In practice, that means they can capture patterns that a fixed low-order chain misses without immediately jumping to a neural serving stack.

Why this family is attractive for Vektor:

- It preserves the stochastic, next-item framing.
- It tends to behave well on discrete symbol sequences.
- It is easier to reason about than transformer embeddings or latent user-state models.
- It gives a realistic path to a bounded native reimplementation if the offline results are strong enough.

Why this family is not a free win:

- The reference implementations are Java-based, so they are better for evaluation than direct production embedding.
- Some of these methods improve accuracy by retaining richer subsequence structure, which can create uncomfortable memory pressure if copied naively.
- Serving characteristics depend heavily on how the data structure is bounded and pruned.

Technical judgment:

If Vektor is going to replace the current first-order Markov model with another stochastic model in the engine, this is the first family worth testing. It offers the best ratio of likely relevance to implementation risk.

### 2. Hidden Markov Models

Open-source reference:

- [pomegranate Hidden Markov Models](https://pomegranate.readthedocs.io/en/latest/tutorials/B_Model_Tutorial_4_Hidden_Markov_Models.html)

An HMM changes the modeling assumption. Instead of saying "the next key depends only on the observed suffix," it says "the system is moving through hidden workload states, and those hidden states generate the observed keys."

That can be useful if your traffic has regime behavior such as:

- startup versus steady state,
- tenant-specific access modes,
- batch jobs or cron windows,
- fan-out sequences that are not obvious from the last one or two keys.

Why an HMM is interesting:

- It is still probabilistic and sequence-aware.
- It gives a cleaner way to model latent operating modes than a plain transition table.
- It can explain why similar observed suffixes sometimes lead to different next keys.

Why it is probably not the first production replacement:

- Inference is materially more expensive than a hash-table lookup on a local transition table.
- Online updating is less natural than simply incrementing transition counts.
- The hidden-state abstraction is valuable only if the trace really contains stable regimes.

Technical judgment:

HMMs are better as an offline baseline or sidecar phase detector than as the first thing to wire into `engine/src/decision_engine.cpp`. If they win, the most realistic production use is likely to be regime classification that informs a cheaper predictor, not a full direct replacement.

### 3. Sequential recommenders that blend order and similarity: `FOSSIL`

Open-source reference:

- [RecBole](https://github.com/RUCAIBox/RecBole)
- [RecBole FOSSIL](https://recbole.io/docs/user_guide/model/sequential/fossil.html)

`FOSSIL` is one of the more interesting recommender-style options because it does not throw away sequential structure. It combines high-order Markov behavior with item similarity, which is a more natural bridge from Vektor's current model than jumping straight to a transformer.

Why it could help:

- Some keys may be substitutable or semantically related even when exact transition counts are sparse.
- High-order information helps when local first-order transitions are too brittle.
- It may outperform pure transition-count models on sparse traces.

Why the fit is conditional:

- Recommender models usually assume an entity boundary such as a user, session, or account.
- If Vektor treats all traffic as a single global stream, the signal may smear together in ways these models were not built for.
- Training and serving are heavier than the current in-engine design.

Technical judgment:

`FOSSIL` is a strong offline benchmark if traces can be segmented by connection, tenant, service, or session. Without that segmentation, its theoretical strengths may not transfer cleanly.

### 4. Deep sequential recommenders: `SASRec`, `GRU4Rec`, `NARM`

Open-source reference:

- [RecBole sequential model catalog](https://recbole.io/docs/recbole/recbole.model.sequential_recommender.html)
- [RecBole SASRec](https://recbole.io/docs/user_guide/model/sequential/sasrec.html)
- [RecBole GRU4Rec](https://recbole.io/docs/user_guide/model/sequential/gru4rec.html)
- [RecBole NARM](https://recbole.io/docs/user_guide/model/sequential/narm.html)

These are useful for answering one question: how much predictive signal exists beyond what a compact stochastic model can exploit?

That is an important question. It is just not the same as asking what should run in the C++ loop.

Why these models are valuable:

- They provide a higher-capacity benchmark for long-range dependencies.
- They help estimate the accuracy ceiling on exported traces.
- They can reveal whether attention over longer histories materially changes top-k recall.

Why they are a poor first production choice:

- serving complexity is much higher,
- online updates are far less natural,
- memory and infrastructure costs move in the wrong direction,
- model behavior becomes harder to inspect and debug under live traffic.

Technical judgment:

Use these models as offline comparators or teacher models. If one of them produces a large accuracy gain, the next step should not be "embed PyTorch into the proxy." The next step should be to ask whether the gain can be distilled into a compact candidate table, re-ranker, or bounded native model.

### 5. Contextual bandit re-ranking

Open-source reference:

- [Vowpal Wabbit](https://github.com/VowpalWabbit/vowpal_wabbit)
- [Vowpal Wabbit contextual bandits](https://vowpalwabbit.org/docs/vowpal_wabbit/python/latest/tutorials/python_Contextual_bandits_and_Vowpal_Wabbit.html)

This is not a sequence model, and that is exactly why it is interesting.

A contextual bandit can sit on top of an existing candidate generator and learn which candidate should be prioritized under a given context. For Vektor, the candidate generator can remain the current Markov chain or a future variable-order successor.

Examples of context that would make this useful:

- tenant or service identity,
- command type,
- shard or backend identity,
- recent hit / miss outcomes,
- time-of-day bucket,
- request burst indicators.

Why this is attractive:

- it preserves a cheap candidate-generation stage,
- it learns from live reward signals,
- it localizes model complexity to ranking rather than generation,
- it offers a safer rollout path than swapping the entire prediction core.

Why it is not sufficient on its own:

- without meaningful context, it has little to work with,
- it still needs a reward definition,
- it does not solve sequence modeling by itself.

Technical judgment:

If Vektor evolves beyond pure key-history features, a contextual bandit re-ranker is one of the most production-realistic upgrades available.

## Recommendation

If the goal is to improve production prediction quality without losing the character of the system, start with variable-order sequence prediction.

That recommendation is less glamorous than "use a transformer," but it is technically stronger.

The reason is straightforward:

- the prediction target stays the same,
- the data type stays the same,
- the serving assumptions stay close to the current design,
- the path from offline win to native implementation is believable.

In concrete terms, the sequence of work should be:

1. Keep the current bounded first-order Markov chain as the baseline.
2. Export representative traces and benchmark `CPT+`, `TDAG`, and `DG` offline.
3. Measure both prediction quality and runtime characteristics.
4. If one of those models wins cleanly, implement the smallest bounded native approximation that preserves the gain.
5. Only then decide whether more complex offline models are worth the operational cost.

If the goal shifts toward adaptation under richer context, keep the stochastic candidate generator and add a bandit re-ranker. That is the highest-leverage hybrid architecture in this space.

If the goal is simply to find the upper bound on attainable accuracy, use RecBole models offline. Treat them as measuring instruments, not deployment targets.

## How To Benchmark In This Repo

The existing bench assets are a good starting point:

- `benchmarks/traces/generate.go`
- `internal/bench/replayer.go`
- `cmd/bench/main.go`

They currently emphasize replay and latency. For model evaluation, extend them or add a parallel harness that measures:

- top-1 accuracy,
- top-k hit rate,
- MRR@k,
- prediction latency per event,
- update latency per event,
- model memory footprint,
- behavior during cold start,
- degradation under high-cardinality key churn.

For honesty, benchmark on more than one trace shape:

- deterministic sequential traces,
- skewed Zipfian traces,
- traces with repeated motifs,
- traces with abrupt phase changes,
- traces segmented by tenant or session if that metadata exists.

The important thing is to avoid proving that a model works only on synthetic patterns it was always going to win on.

## Bottom Line

There are many open-source models that can "perform handling predictions" if the phrase is taken loosely enough. For Vektor, that is not a useful bar.

The useful bar is this:

- Does the model improve next-key prediction on real traces?
- Can it update online or near-online?
- Can it run without turning the proxy into an inference service?
- Can the team explain, bound, and operate it under production load?

Against that bar, the best near-term candidate is a variable-order stochastic sequence model such as `CPT+` or a related predictor in the SPMF family.

The best hybrid path is Markov-style candidate generation plus contextual-bandit re-ranking with Vowpal Wabbit.

The best way to explore the outer accuracy limit is offline benchmarking with RecBole models such as `FOSSIL`, `SASRec`, `GRU4Rec`, and `NARM`.

## Sources

- [SPMF sequence prediction comparison](https://www.philippe-fournier-viger.com/spmf/CompareSequencePredictionModels.php)
- [SPMF library overview](https://www.philippe-fournier-viger.com/spmf/)
- [pomegranate Hidden Markov Models](https://pomegranate.readthedocs.io/en/latest/tutorials/B_Model_Tutorial_4_Hidden_Markov_Models.html)
- [RecBole repository](https://github.com/RUCAIBox/RecBole)
- [RecBole sequential model catalog](https://recbole.io/docs/recbole/recbole.model.sequential_recommender.html)
- [RecBole FOSSIL](https://recbole.io/docs/user_guide/model/sequential/fossil.html)
- [RecBole SASRec](https://recbole.io/docs/user_guide/model/sequential/sasrec.html)
- [RecBole GRU4Rec](https://recbole.io/docs/user_guide/model/sequential/gru4rec.html)
- [RecBole NARM](https://recbole.io/docs/user_guide/model/sequential/narm.html)
- [Vowpal Wabbit repository](https://github.com/VowpalWabbit/vowpal_wabbit)
- [Vowpal Wabbit contextual bandits](https://vowpalwabbit.org/docs/vowpal_wabbit/python/latest/tutorials/python_Contextual_bandits_and_Vowpal_Wabbit.html)
