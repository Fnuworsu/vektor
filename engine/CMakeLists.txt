cmake_minimum_required(VERSION 3.10)
project(vektor_engine CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

include_directories(include)

add_library(vektor_engine SHARED src/ring_buffer.cpp)
if(APPLE)
    set_target_properties(vektor_engine PROPERTIES MACOSX_RPATH TRUE)
endif()

set(THREADS_PREFER_PTHREAD_FLAG ON)
find_package(Threads REQUIRED)
target_link_libraries(vektor_engine PRIVATE Threads::Threads)

add_executable(ring_buffer_test tests/ring_buffer_test.cpp)
target_link_libraries(ring_buffer_test PRIVATE vektor_engine Threads::Threads)

enable_testing()
add_test(NAME ring_buffer_test COMMAND ring_buffer_test)
