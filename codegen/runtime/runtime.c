// Xuesos++ Runtime Library
// This file provides the runtime support for compiled Xuesos++ programs.

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <math.h>
#include <stdint.h>
#include <stdbool.h>

// --- String type ---

typedef struct {
    char *data;
    int64_t len;
    int64_t cap;
} XppString;

XppString xpp_string_new(const char *s) {
    XppString str;
    str.len = strlen(s);
    str.cap = str.len + 1;
    str.data = (char *)malloc(str.cap);
    memcpy(str.data, s, str.len + 1);
    return str;
}

XppString xpp_string_concat(XppString a, XppString b) {
    XppString result;
    result.len = a.len + b.len;
    result.cap = result.len + 1;
    result.data = (char *)malloc(result.cap);
    memcpy(result.data, a.data, a.len);
    memcpy(result.data + a.len, b.data, b.len + 1);
    return result;
}

bool xpp_string_eq(XppString a, XppString b) {
    if (a.len != b.len) return false;
    return memcmp(a.data, b.data, a.len) == 0;
}

int64_t xpp_string_len(XppString s) {
    return s.len;
}

void xpp_string_free(XppString *s) {
    if (s->data) {
        free(s->data);
        s->data = NULL;
    }
}

// --- Array type ---

typedef struct {
    void **data;
    int64_t len;
    int64_t cap;
} XppArray;

XppArray xpp_array_new(int64_t cap) {
    XppArray arr;
    arr.len = 0;
    arr.cap = cap > 0 ? cap : 8;
    arr.data = (void **)malloc(sizeof(void *) * arr.cap);
    return arr;
}

void xpp_array_push(XppArray *arr, void *elem) {
    if (arr->len >= arr->cap) {
        arr->cap *= 2;
        arr->data = (void **)realloc(arr->data, sizeof(void *) * arr->cap);
    }
    arr->data[arr->len++] = elem;
}

void *xpp_array_get(XppArray *arr, int64_t index) {
    if (index < 0 || index >= arr->len) {
        fprintf(stderr, "runtime error: index %lld out of bounds (len=%lld)\n",
                (long long)index, (long long)arr->len);
        exit(1);
    }
    return arr->data[index];
}

void xpp_array_set(XppArray *arr, int64_t index, void *value) {
    if (index < 0 || index >= arr->len) {
        fprintf(stderr, "runtime error: index %lld out of bounds (len=%lld)\n",
                (long long)index, (long long)arr->len);
        exit(1);
    }
    arr->data[index] = value;
}

// --- Print functions ---

void xpp_print_int(int64_t val) {
    printf("%lld\n", (long long)val);
}

void xpp_print_float(double val) {
    printf("%g\n", val);
}

void xpp_print_string(XppString val) {
    printf("%.*s\n", (int)val.len, val.data);
}

void xpp_print_bool(bool val) {
    printf("%s\n", val ? "xuitru" : "xuinia");
}

void xpp_print_char(char val) {
    printf("%c\n", val);
}
