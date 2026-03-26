# Prediction Model Research

## Problem Framing

Vektor is currently a bounded first-order Markov predictor running in the C++ engine hot path. That makes the relevant question narrower than "best open-source model":

- The job is next-key sequence prediction, not general forecasting.
- The current engine is optimized for tiny per-event cost and bounded memory.
- Any replacement has to justify additional latency, memory, and integration complexity.

The best candidates are the ones that either:

1. preserve the stochastic / sequence-local properties of the current model while improving coverage, or
2. stay off the hot path and re-rank / distill predictions into something lightweight.

## Shortlist

| Model | Open-source option | Why it fits Vektor | Fit for hot path | Main tradeoff |
| --- | --- | --- | --- | --- |
| Variable-order sequence prediction (`CPT+`, `TDAG`, `DG`) | [SPMF](https://www.philippe-fournier-viger.com/spmf/) | Closest upgrade path from the current first-order Markov chain. These models are built for next-symbol / next-item sequence prediction and explicitly compare accuracy, coverage, training time, and prediction time. | Medium | Best conceptual fit, but the reference implementation is Java, so production use likely means sidecar evaluation, offline benchmarking, or a C++ reimplementation of the winning idea. |
| Hidden Markov Model (`DenseHMM`) | [pomegranate](https://pomegranate.readthedocs.io/en/latest/tutorials/B_Model_Tutorial_4_Hidden_Markov_Models.html) | Good when key-access traffic moves through hidden workload phases such as warm-up, fan-out, cron bursts, or tenant-specific modes. | Low | Strong probabilistic baseline, but usually better as an offline / sidecar phase detector than an inline per-request scorer. |
| High-order sparse sequential recommender (`FOSSIL`) | [RecBole](https://recbole.io/docs/user_guide/model/sequential/fossil.html) | Explicitly mixes high-order Markov structure with item similarity and is designed for sparse sequential data. | Low | Useful if you can identify a tenant / connection / service identity. Less natural if all traffic is treated as one global stream. |
| Self-attention sequential recommender (`SASRec`) | [RecBole](https://recbole.io/docs/user_guide/model/sequential/sasrec.html) | Strong offline benchmark when longer-range dependencies matter and first-order transitions lose signal. | Low | Accuracy can improve, but serving cost and Python / GPU-centric tooling make it a poor direct hot-path replacement. |
| Session RNN / attention (`GRU4Rec`, `NARM`) | [RecBole GRU4Rec](https://recbole.io/docs/user_guide/model/sequential/gru4rec.html), [RecBole NARM](https://recbole.io/docs/user_guide/model/sequential/narm.html) | Useful session-based baselines for shorter traces and bursty request streams. | Low | Better as offline comparators than production inference inside this C++ loop. |
| Contextual bandit re-ranker | [Vowpal Wabbit](https://vowpalwabbit.org/docs/vowpal_wabbit/python/latest/tutorials/python_Contextual_bandits_and_Vowpal_Wabbit.html) | Good if Vektor starts emitting context such as tenant, command type, miss / hit history, shard, or time bucket. Works well as a re-ranker over existing candidates instead of replacing the sequence model. | Medium | Not a sequence model by itself. It needs useful features and a reward signal. |

## Recommendation

### Best immediate stochastic upgrade

Start with a variable-order sequence predictor, not a neural model.

Why:

- It is the closest match to the current bounded Markov design.
- It keeps the prediction objective identical: "given the recent key history, what key comes next?"
- It is much more likely to preserve Vektor's latency profile than an RNN or transformer.

The most relevant family is:

- `CPT+`
- `TDAG`
- `DG`

SPMF already exposes a comparison harness for these sequence predictors, which makes it the fastest way to learn whether a higher-order stochastic model beats the current first-order baseline on Vektor traces.

### Best augmentation path

If you can add context, keep the Markov candidate generator and add a contextual bandit re-ranker with Vowpal Wabbit.

That design is pragmatic because:

- candidate generation stays cheap and deterministic,
- re-ranking can learn from hit / miss outcomes online,
- rollout risk is lower than swapping the core predictor outright.

This is the most production-friendly path if you want the model to adapt to tenant-specific or workload-specific drift without introducing a heavyweight inference stack.

### Best offline accuracy benchmark

Use RecBole to benchmark `FOSSIL`, `SASRec`, `GRU4Rec`, and `NARM` offline against trace exports.

This is worth doing to find the accuracy ceiling, but these models should be treated as:

- offline evaluators,
- teacher models for distillation,
- or sidecar scorers that periodically export compact top-k transition tables.

They are not good first candidates for direct insertion into `engine/src/decision_engine.cpp`.

## What To Benchmark In This Repo

The repo already has a trace generator and replay harness:

- `benchmarks/traces/generate.go`
- `internal/bench/replayer.go`
- `cmd/bench/main.go`

Extend evaluation around those assets with metrics that reflect prediction quality, not just Redis latency:

- top-1 hit rate
- top-k hit rate
- MRR@k
- prediction latency per event
- model memory footprint
- state / parameter count
- cold-start behavior

## Practical Rollout Order

1. Keep the current bounded first-order Markov chain as the baseline.
2. Prototype a variable-order predictor offline using SPMF sequence models on exported traces.
3. If it wins on hit rate without exploding memory, implement a bounded C++ version of the winning design.
4. If context features become available, add Vowpal Wabbit as a re-ranker over Markov-produced candidates.
5. Use RecBole models only to determine the upside of more complex sequential modeling.

## Decision Summary

- If the goal is a realistic production successor to the current stochastic engine: use a `CPT+` / variable-order Markov style benchmark first.
- If the goal is online adaptation with side information: use Vowpal Wabbit as a re-ranker.
- If the goal is to find the theoretical accuracy ceiling: benchmark `FOSSIL`, `SASRec`, `GRU4Rec`, and `NARM` offline via RecBole.

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
