#include "markov_chain.h"
#include <algorithm>

MarkovChain::MarkovChain(int order, size_t max_keys)
    : order_(order), max_keys_(max_keys) {}

std::string MarkovChain::join(const std::vector<std::string>& history) const {
    std::string result;
    size_t start = (history.size() > static_cast<size_t>(order_)) 
                   ? history.size() - order_ : 0;
    
    for (size_t i = start; i < history.size(); ++i) {
        result += history[i];
        if (i < history.size() - 1) {
            result += "|";
        }
    }
    return result;
}

void MarkovChain::observe(const std::vector<std::string>& history, const std::string& next_key) {
    if (history.empty()) return;
    
    std::string state_key = join(history);

    if (states_.find(state_key) == states_.end() && states_.size() >= max_keys_) {
        std::string lfu_key;
        uint64_t min_count = UINT64_MAX;
        
        for (const auto& pair : states_) {
            if (pair.second.access_count < min_count) {
                min_count = pair.second.access_count;
                lfu_key = pair.first;
            }
        }
        
        if (!lfu_key.empty()) {
            states_.erase(lfu_key);
        }
    }

    auto& entry = states_[state_key];
    entry.transitions[next_key]++;
    entry.access_count++;
}

std::vector<Prediction> MarkovChain::predict(const std::vector<std::string>& history) const {
    if (history.empty()) return {};

    std::string state_key = join(history);
    auto it = states_.find(state_key);
    if (it == states_.end()) return {};

    uint64_t total = 0;
    for (const auto& pair : it->second.transitions) {
        total += pair.second;
    }

    std::vector<Prediction> predictions;
    if (total == 0) return predictions;

    for (const auto& pair : it->second.transitions) {
        predictions.push_back({pair.first, static_cast<double>(pair.second) / total});
    }

    std::sort(predictions.begin(), predictions.end(), 
              [](const Prediction& a, const Prediction& b) {
                  return a.probability > b.probability;
              });

    return predictions;
}

size_t MarkovChain::size() const {
    return states_.size();
}
