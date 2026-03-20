#include "ring_buffer.h"
#include <iostream>
#include <thread>
#include <cstddef>
#include <cassert>

void test_single_threaded() {
    RingBuffer<int, 4> rb;
    assert(rb.Capacity() == 4);
    assert(rb.Size() == 0);

    assert(rb.push(1));
    assert(rb.push(2));
    assert(rb.Size() == 2);

    int val = 0;
    assert(rb.pop(&val));
    assert(val == 1);
    assert(rb.pop(&val));
    assert(val == 2);
    
    assert(!rb.pop(&val));
}

void test_full_buffer() {
    RingBuffer<int, 2> rb;
    assert(rb.push(10));
    assert(rb.push(20));
    assert(!rb.push(30));
    assert(rb.Size() == 2);
}

void test_empty_buffer() {
    RingBuffer<int, 2> rb;
    int val = 0;
    assert(!rb.pop(&val));
}

void test_concurrent() {
    RingBuffer<int, 1024> rb;
    const int num_items = 1000000;

    std::thread producer([&]() {
        for (int i = 0; i < num_items; ++i) {
            while (!rb.push(i)) {}
        }
    });

    std::thread consumer([&]() {
        for (int i = 0; i < num_items; ++i) {
            int val = 0;
            while (!rb.pop(&val)) {}
            assert(val == i);
        }
    });

    producer.join();
    consumer.join();
}

void test_cache_line() {
    using IntRingBuffer = RingBuffer<int, 8>;
    size_t head_offset = offsetof(IntRingBuffer, head_);
    size_t tail_offset = offsetof(IntRingBuffer, tail_);
    size_t diff = (head_offset > tail_offset) ? (head_offset - tail_offset) : (tail_offset - head_offset);
    assert(diff >= 64);
}

int main() {
    test_single_threaded();
    test_full_buffer();
    test_empty_buffer();
    test_cache_line();
    test_concurrent();
    std::cout << "All ring buffer tests passed!\n";
    return 0;
}
