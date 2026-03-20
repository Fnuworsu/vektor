#ifndef VEKTOR_ENGINE_H
#define VEKTOR_ENGINE_H

#include <stdint.h>
#include <stddef.h>

#ifdef __cplusplus
extern "C" {
#endif

typedef struct vektor_engine_t vektor_engine_t;

vektor_engine_t* vektor_engine_create(int markov_order, int max_keys, double threshold);

void vektor_engine_destroy(vektor_engine_t* engine);

int vektor_engine_push_event(vektor_engine_t* engine, const char* key, int64_t timestamp_ns);

void vektor_engine_set_callback(vektor_engine_t* engine, void(*cb)(const char* key, double prob, void* userdata), void* userdata);

void vektor_engine_start(vektor_engine_t* engine);

void vektor_engine_stop(vektor_engine_t* engine);

size_t vektor_engine_get_tracked_keys(vektor_engine_t* engine);

#ifdef __cplusplus
}
#endif

#endif
