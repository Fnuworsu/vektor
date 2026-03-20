#pragma once

#include <atomic>
#include <cstddef>
#include <type_traits>
#include <cassert>

template <typename T, size_t N>
class RingBuffer {
    static_assert(N > 0 && (N & (N - 1)) == 0, "");

public:
    RingBuffer() : head_(0), tail_(0) {}

    bool push(const T& item) {
        size_t current_tail = tail_.load(std::memory_order_relaxed);
        size_t current_head = head_.load(std::memory_order_acquire);
        
        if (current_tail - current_head == N) {
            return false; 
        }

        buffer_[current_tail & (N - 1)] = item;
        tail_.store(current_tail + 1, std::memory_order_release);
        return true;
    }

    bool pop(T* item) {
        size_t current_head = head_.load(std::memory_order_relaxed);
        size_t current_tail = tail_.load(std::memory_order_acquire);

        if (current_head == current_tail) {
            return false; 
        }

        *item = buffer_[current_head & (N - 1)];
        head_.store(current_head + 1, std::memory_order_release);
        return true;
    }

    size_t Size() const {
        return tail_.load(std::memory_order_relaxed) - head_.load(std::memory_order_relaxed);
    }

    size_t Capacity() const {
        return N;
    }

    T buffer_[N];

    alignas(64) std::atomic<size_t> tail_;
    alignas(64) std::atomic<size_t> head_;
};
