#include "markov_chain.h"
#include <iostream>
#include <cassert>
#include <cmath>

void test_basic_probabilities() {
    MarkovChain mc(2, 100);
    std::vector<std::string> hist = {"A", "B"};
    mc.observe(hist, "C");
    mc.observe(hist, "C");
    mc.observe(hist, "C");
    mc.observe(hist, "D");

    auto preds = mc.predict(hist);
    assert(preds.size() == 2);
    
    assert(preds[0].key == "C");
    assert(std::abs(preds[0].probability - 0.75) < 1e-6);
    
    assert(preds[1].key == "D");
    assert(std::abs(preds[1].probability - 0.25) < 1e-6);
    
    double total_prob = preds[0].probability + preds[1].probability;
    assert(std::abs(total_prob - 1.0) < 1e-6);
}

void test_sequence_order() {
    MarkovChain mc(2, 100);
    mc.observe({"A"}, "B");
    mc.observe({"A", "B"}, "C");
    mc.observe({"B", "C"}, "A");
    mc.observe({"C", "A"}, "B");
    mc.observe({"A", "B"}, "D");

    auto preds = mc.predict({"A", "B"});
    assert(preds.size() == 2);
    assert(preds[0].key == "C" || preds[0].key == "D");
    assert(preds[0].probability == 0.5);
    assert(preds[1].probability == 0.5);
}

void test_eviction() {
    MarkovChain mc(1, 4);
    mc.observe({"A"}, "1");
    mc.observe({"A"}, "1");
    mc.observe({"A"}, "1");
    mc.observe({"A"}, "1");
    
    mc.observe({"B"}, "2");
    mc.observe({"B"}, "2");
    mc.observe({"B"}, "2");

    mc.observe({"D"}, "4");
    mc.observe({"D"}, "4");
    
    mc.observe({"C"}, "3");
    
    assert(mc.size() == 4);
    
    mc.observe({"E"}, "5");
    
    assert(mc.size() == 4);
    assert(mc.predict({"C"}).empty());
    assert(mc.predict({"E"}).size() == 1);
}

void test_empty_history() {
    MarkovChain mc(2, 100);
    assert(mc.predict({}).empty());
}

int main() {
    test_basic_probabilities();
    test_sequence_order();
    test_eviction();
    test_empty_history();
    std::cout << "All markov chain tests passed!\n";
    return 0;
}
