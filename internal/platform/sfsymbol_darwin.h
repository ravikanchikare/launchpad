#pragma once

#include <stddef.h>

// Returns malloc'd PNG bytes for a system symbol, or NULL on failure.
void *HarnezPadSFSymbolPNG(const char *name, int pointSize, size_t *outLen);
