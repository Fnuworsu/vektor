#include "decision_engine.h"
#include "engine.h"
#include <iostream>
#include <cassert>
#include <vector>
#include <string>
#include <thread>
#include <chrono>
#include <mutex>
#include <memory>

struct CallbackRecord {
    std::string key;
    double prob;
};

struct TestContext {
    std::vector<CallbackRecord> records;
    std::mutex mtx;
};

void test_callback(const char* key, double prob, void* userdata) {
    auto* ctx = static_cast<TestContext*>(userdata);
    std::lock_guard<std::mutex> lock(ctx->mtx);
    ctx->records.push_back({std::string(key), prob});
}

void test_decision_engine() {
    auto de = std::make_unique<DecisionEngine>(1, 100, 0.6);
    TestContext ctx;
    de->set_callback(test_callback, &ctx);
    
    de->start();

    assert(de->push_event("A", 1000) == 0);
    assert(de->push_event("B", 2000) == 0);
    assert(de->push_event("A", 3000) == 0);
    assert(de->push_event("B", 4000) == 0);
    assert(de->push_event("A", 5000) == 0);
    assert(de->push_event("B", 6000) == 0);

    for (int i = 0; i < 100; i++) {
        size_t size = 0;
        {
            std::lock_guard<std::mutex> lock(ctx.mtx);
            size = ctx.records.size();
        }
        if (size >= 2) break;
        std::this_thread::sleep_for(std::chrono::milliseconds(10));
    }

    de->stop();

    bool found_b = false;
    for (const auto& r : ctx.records) {
        if (r.key == "B" && r.prob > 0.6) {
            found_b = true;
            break;
        }
    }
    assert(found_b);
}

void test_c_api() {
    vektor_engine_t* engine = vektor_engine_create(1, 100, 0.6);
    assert(engine != nullptr);

    TestContext ctx;
    vektor_engine_set_callback(engine, test_callback, &ctx);
    
    vektor_engine_start(engine);

    assert(vektor_engine_push_event(engine, "A", 1000) == 0);
    assert(vektor_engine_push_event(engine, "B", 2000) == 0);
    assert(vektor_engine_push_event(engine, "A", 3000) == 0);
    assert(vektor_engine_push_event(engine, "B", 4000) == 0);
    assert(vektor_engine_push_event(engine, "A", 5000) == 0);
    assert(vektor_engine_push_event(engine, "B", 6000) == 0);

    for (int i = 0; i < 100; i++) {
        size_t size = 0;
        {
            std::lock_guard<std::mutex> lock(ctx.mtx);
            size = ctx.records.size();
        }
        if (size >= 2) break;
        std::this_thread::sleep_for(std::chrono::milliseconds(10));
    }

    vektor_engine_stop(engine);
    vektor_engine_destroy(engine);

    bool found_b = false;
    for (const auto& r : ctx.records) {
        if (r.key == "B" && r.prob > 0.6) {
            found_b = true;
            break;
        }
    }
    assert(found_b);
}

int main() {
    test_decision_engine();
    test_c_api();
    std::cout << "All decision engine tests passed!\n";
    return 0;
}
