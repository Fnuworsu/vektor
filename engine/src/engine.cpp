#include "engine.h"
#include "decision_engine.h"

extern "C" {

vektor_engine_t* vektor_engine_create(int markov_order, int max_keys, double threshold) {
    return reinterpret_cast<vektor_engine_t*>(new DecisionEngine(markov_order, max_keys, threshold));
}

void vektor_engine_destroy(vektor_engine_t* engine) {
    if (engine) {
        delete reinterpret_cast<DecisionEngine*>(engine);
    }
}

int vektor_engine_push_event(vektor_engine_t* engine, const char* key, int64_t timestamp_ns) {
    if (!engine) return 1;
    return reinterpret_cast<DecisionEngine*>(engine)->push_event(key, timestamp_ns);
}

void vektor_engine_set_callback(vektor_engine_t* engine, void(*cb)(const char* key, double prob, void* userdata), void* userdata) {
    if (!engine) return;
    reinterpret_cast<DecisionEngine*>(engine)->set_callback(cb, userdata);
}

void vektor_engine_start(vektor_engine_t* engine) {
    if (!engine) return;
    reinterpret_cast<DecisionEngine*>(engine)->start();
}

void vektor_engine_stop(vektor_engine_t* engine) {
    if (!engine) return;
    reinterpret_cast<DecisionEngine*>(engine)->stop();
}

}
