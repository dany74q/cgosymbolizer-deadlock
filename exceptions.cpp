#include <string>
#include <stdexcept>

extern "C" {

void throwAndCatch() {
    try {
        throw std::runtime_error("pasten");
    } catch (std::runtime_error&) {
    }
}

}