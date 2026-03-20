#pragma once

#include <string>
#include <vector>
#include <unordered_map>
#include <cstdint>

struct Prediction {
    std::string key;
    double probability;
};

class MarkovChain {
public:
    MarkovChain(int order, size_t max_keys);

    void observe(const std::vector<std::string>& history, const std::string& next_key);
    std::vector<Prediction> predict(const std::vector<std::string>& history) const;
    size_t size() const;

private:
    std::string join(const std::vector<std::string>& history) const;

    int order_;
    size_t max_keys_;

    using TransitionTable = std::unordered_map<std::string, uint64_t>;
    
    struct StateEntry {
        TransitionTable transitions;
        uint64_t access_count;
    };

    std::unordered_map<std::string, StateEntry> states_;
};
