#include "decision_engine.h"

DecisionEngine::DecisionEngine(int markov_order, int max_keys, double threshold)
    : markov_order_(markov_order), threshold_(threshold),
      markov_chain_(markov_order, max_keys),
      callback_(nullptr), userdata_(nullptr),
      running_(false) {
}

DecisionEngine::~DecisionEngine() {
    stop();
}

int DecisionEngine::push_event(const char* key, int64_t timestamp_ns) {
    AccessEvent event;
    std::strncpy(event.key, key, sizeof(event.key) - 1);
    event.key[sizeof(event.key) - 1] = '\0';
    event.timestamp_ns = timestamp_ns;
    
    if (ring_buffer_.push(event)) {
        return 0;
    }
    return 1;
}

void DecisionEngine::set_callback(PrefetchCallback cb, void* userdata) {
    callback_ = cb;
    userdata_ = userdata;
}

void DecisionEngine::start() {
    bool expected = false;
    if (running_.compare_exchange_strong(expected, true)) {
        worker_ = std::thread(&DecisionEngine::loop, this);
    }
}

void DecisionEngine::stop() {
    bool expected = true;
    if (running_.compare_exchange_strong(expected, false)) {
        if (worker_.joinable()) {
            worker_.join();
        }
    }
}

void DecisionEngine::loop() {
    AccessEvent event;
    while (running_.load(std::memory_order_relaxed) || ring_buffer_.Size() > 0) {
        if (ring_buffer_.pop(&event)) {
            std::string key_str(event.key);
            
            markov_chain_.observe(history_, key_str);
            
            history_.push_back(key_str);
            if (history_.size() > static_cast<size_t>(markov_order_)) {
                history_.erase(history_.begin());
            }

            auto predictions = markov_chain_.predict(history_);
            for (const auto& pred : predictions) {
                if (pred.probability >= threshold_) {
                    if (callback_) {
                        callback_(pred.key.c_str(), pred.probability, userdata_);
                    }
                }
            }
        } else {
            std::this_thread::yield();
        }
    }
}
