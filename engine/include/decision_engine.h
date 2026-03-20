#pragma once

#include "ring_buffer.h"
#include "markov_chain.h"
#include <thread>
#include <atomic>
#include <string>
#include <cstring>
#include <vector>

struct AccessEvent {
    char key[256];
    int64_t timestamp_ns;
};

using PrefetchCallback = void(*)(const char* key, double prob, void* userdata);

class DecisionEngine {
public:
    DecisionEngine(int markov_order, int max_keys, double threshold);
    ~DecisionEngine();

    int push_event(const char* key, int64_t timestamp_ns);
    void set_callback(PrefetchCallback cb, void* userdata);
    void start();
    void stop();

private:
    void loop();

    int markov_order_;
    double threshold_;
    
    RingBuffer<AccessEvent, 65536> ring_buffer_;
    MarkovChain markov_chain_;
    
    PrefetchCallback callback_;
    void* userdata_;

    std::atomic<bool> running_;
    std::thread worker_;
    std::vector<std::string> history_;
};
