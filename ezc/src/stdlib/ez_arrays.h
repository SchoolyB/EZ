/*
 * ez_arrays.h - arrays module for EZ
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef EZ_ARRAYS_H
#define EZ_ARRAYS_H

#include "../runtime/ez_runtime.h"
#include "../runtime/ez_array.h"

/* Modification */

/*@man append
 *@module arrays
 *@group Modification
 *@sig append(&arr [T], value T)
 *@desc Appends value to the end of arr. Modifies the array in place.
 *@example
 *   import @arrays
 *   mut nums [int] = [1, 2, 3]
 *   arrays.append(nums, 4)
 *   println(nums)
 *@end
 */
void ez_arrays_append(EzArena *arena, EzArray *arr, const void *value);

/*@man insert_at
 *@module arrays
 *@group Modification
 *@sig insert_at(&arr [T], index int, value T)
 *@desc Inserts value at the given index, shifting subsequent elements right. Modifies the array in place.
 *@example
 *   import @arrays
 *   mut nums [int] = [1, 2, 3]
 *   arrays.insert_at(nums, 1, 99)
 *   println(nums)
 *@end
 */
void ez_arrays_insert_at(EzArena *arena, EzArray *arr, int32_t index, const void *value);

/*@man prepend
 *@module arrays
 *@group Modification
 *@sig prepend(&arr [T], value T)
 *@desc Inserts value at the front of arr, shifting all elements right. Modifies the array in place.
 *@example
 *   import @arrays
 *   mut nums [int] = [2, 3, 4]
 *   arrays.prepend(nums, 1)
 *   println(nums)
 *@end
 */
void ez_arrays_prepend(EzArena *arena, EzArray *arr, const void *value);

/*@man remove_at
 *@module arrays
 *@group Modification
 *@sig remove_at(&arr [T], index int)
 *@desc Removes the element at the given index, shifting subsequent elements left. Modifies the array in place.
 *@example
 *   import @arrays
 *   mut nums [int] = [1, 2, 3]
 *   arrays.remove_at(nums, 1)
 *   println(nums)
 *@end
 */
void ez_arrays_remove_at(EzArray *arr, int32_t index);

/*@man remove
 *@module arrays
 *@group Modification
 *@sig remove(&arr [T], value T)
 *@desc Removes the first occurrence of value from arr. Modifies the array in place. Does nothing if value is not found.
 *@example
 *   import @arrays
 *   mut nums [int] = [1, 2, 3, 2]
 *   arrays.remove(nums, 2)
 *   println(nums)
 *@end
 */
void ez_arrays_remove_int(EzArray *arr, int64_t value);
void ez_arrays_remove_float(EzArray *arr, double value);
void ez_arrays_remove_str(EzArray *arr, EzString value);

/*@man clear
 *@module arrays
 *@group Modification
 *@sig clear(&arr [T])
 *@desc Removes all elements from arr, leaving it empty. Modifies the array in place.
 *@example
 *   import @arrays
 *   mut nums [int] = [1, 2, 3]
 *   arrays.clear(nums)
 *   println(nums)
 *@end
 */
void ez_arrays_clear(EzArray *arr);

/*@man fill
 *@module arrays
 *@group Modification
 *@sig fill(&arr [T], value T, count int)
 *@desc Appends count copies of value to arr. Modifies the array in place.
 *@example
 *   import @arrays
 *   mut nums [int] = []
 *   arrays.fill(nums, 0, 5)
 *   println(nums)
 *@end
 */
void ez_arrays_fill(EzArena *arena, EzArray *arr, const void *value, int32_t count);

/* Access — typed via codegen cast */

/*@man get_first
 *@module arrays
 *@group Access
 *@sig get_first(arr [T]) -> T
 *@desc Returns the first element of arr. Panics if arr is empty.
 *@example
 *   import @arrays
 *   mut nums [int] = [10, 20, 30]
 *   println(arrays.get_first(nums))
 *@end
 */
void *ez_arrays_first_ptr(EzArray *arr);

/*@man get_last
 *@module arrays
 *@group Access
 *@sig get_last(arr [T]) -> T
 *@desc Returns the last element of arr. Panics if arr is empty.
 *@example
 *   import @arrays
 *   mut nums [int] = [10, 20, 30]
 *   println(arrays.get_last(nums))
 *@end
 */
void *ez_arrays_last_ptr(EzArray *arr);

/*@man remove_first
 *@module arrays
 *@group Modification
 *@sig remove_first(&arr [T]) -> T
 *@desc Removes and returns the first element of arr. Panics if arr is empty. Modifies the array in place.
 *@example
 *   import @arrays
 *   mut nums [int] = [10, 20, 30]
 *   mut val int = arrays.remove_first(nums)
 *   println(val)
 *@end
 */
void  ez_arrays_remove_first_raw(EzArray *arr, void *out);

/*@man remove_last
 *@module arrays
 *@group Modification
 *@sig remove_last(&arr [T]) -> T
 *@desc Removes and returns the last element of arr. Panics if arr is empty. Modifies the array in place.
 *@example
 *   import @arrays
 *   mut nums [int] = [10, 20, 30]
 *   mut val int = arrays.remove_last(nums)
 *   println(val)
 *@end
 */
void  ez_arrays_remove_last_raw(EzArray *arr, void *out);

/* Access — int64_t variants for internal use */
int64_t ez_arrays_get_first(EzArray *arr);
int64_t ez_arrays_get_last(EzArray *arr);
int64_t ez_arrays_remove_last(EzArray *arr);
int64_t ez_arrays_remove_first(EzArray *arr);

/* Query */

/*@man is_empty
 *@module arrays
 *@group Query
 *@sig is_empty(arr [T]) -> bool
 *@desc Returns true if arr has no elements.
 *@example
 *   import @arrays
 *   mut nums [int] = []
 *   println(arrays.is_empty(nums))
 *@end
 */
bool ez_arrays_is_empty(EzArray *arr);

/*@man contains
 *@module arrays
 *@group Query
 *@sig contains(arr [T], value T) -> bool
 *@desc Returns true if arr contains value.
 *@example
 *   import @arrays
 *   mut nums [int] = [1, 2, 3]
 *   println(arrays.contains(nums, 2))
 *@end
 */
bool ez_arrays_contains_int(EzArray *arr, int64_t value);
bool ez_arrays_contains_float(EzArray *arr, double value);
bool ez_arrays_contains_str(EzArray *arr, EzString value);

/*@man index_of
 *@module arrays
 *@group Query
 *@sig index_of(arr [T], value T) -> int
 *@desc Returns the index of the first occurrence of value in arr, or -1 if not found.
 *@example
 *   import @arrays
 *   mut nums [int] = [10, 20, 30]
 *   println(arrays.index_of(nums, 20))
 *@end
 */
int64_t ez_arrays_index_of_int(EzArray *arr, int64_t value);
int64_t ez_arrays_index_of_str(EzArray *arr, EzString value);

/*@man count
 *@module arrays
 *@group Query
 *@sig count(arr [T], value T) -> int
 *@desc Returns the number of times value appears in arr.
 *@example
 *   import @arrays
 *   mut nums [int] = [1, 2, 2, 3]
 *   println(arrays.count(nums, 2))
 *@end
 */
int64_t ez_arrays_count(EzArray *arr, int64_t value);

/*@man is_equal
 *@module arrays
 *@group Query
 *@sig is_equal(a [T], b [T]) -> bool
 *@desc Returns true if a and b have the same length and identical elements. T must be a primitive or string. Use this instead of == which is not allowed on arrays.
 *@example
 *   import @arrays
 *   mut a [int] = [1, 2, 3]
 *   mut b [int] = [1, 2, 3]
 *   println(arrays.is_equal(a, b))
 *@end
 */
bool ez_arrays_is_equal_prim(EzArray *a, EzArray *b);
bool ez_arrays_is_equal_str(EzArray *a, EzArray *b);

/* Transformation */

/*@man reverse
 *@module arrays
 *@group Transformation
 *@sig reverse(arr [T]) -> [T]
 *@desc Returns a new array with elements in reverse order. Does not modify the original.
 *@example
 *   import @arrays
 *   mut nums [int] = [1, 2, 3]
 *   println(arrays.reverse(nums))
 *@end
 */
EzArray ez_arrays_reverse(EzArena *arena, EzArray *arr);

/*@man slice
 *@module arrays
 *@group Transformation
 *@sig slice(arr [T], start int, end int) -> [T]
 *@desc Returns a new array containing elements from index start (inclusive) to end (exclusive). Does not modify the original.
 *@example
 *   import @arrays
 *   mut nums [int] = [10, 20, 30, 40]
 *   println(arrays.slice(nums, 1, 3))
 *@end
 */
EzArray ez_arrays_slice(EzArena *arena, EzArray *arr, int32_t start, int32_t end);

/*@man concat
 *@module arrays
 *@group Transformation
 *@sig concat(a [T], b [T]) -> [T]
 *@desc Returns a new array containing all elements of a followed by all elements of b.
 *@example
 *   import @arrays
 *   mut a [int] = [1, 2]
 *   mut b [int] = [3, 4]
 *   println(arrays.concat(a, b))
 *@end
 */
EzArray ez_arrays_concat(EzArena *arena, EzArray *a, EzArray *b);

/*@man deduplicate
 *@module arrays
 *@group Transformation
 *@sig deduplicate(arr [T]) -> [T]
 *@desc Returns a new array with duplicate values removed, preserving the order of first occurrence.
 *@example
 *   import @arrays
 *   mut nums [int] = [1, 2, 2, 3, 1]
 *   println(arrays.deduplicate(nums))
 *@end
 */
EzArray ez_arrays_deduplicate(EzArena *arena, EzArray *arr);

/*@man flatten
 *@module arrays
 *@group Transformation
 *@sig flatten(arr [[T]]) -> [T]
 *@desc Flattens one level of nesting, returning a single array of all inner elements.
 *@example
 *   import @arrays
 *   mut nested [[int]] = [[1, 2], [3, 4]]
 *   println(arrays.flatten(nested))
 *@end
 */
EzArray ez_arrays_flatten(EzArena *arena, EzArray *arr);

/*@man split_every
 *@module arrays
 *@group Transformation
 *@sig split_every(arr [T], size int) -> [[T]]
 *@desc Splits arr into sub-arrays of at most size elements each. The last chunk may be smaller.
 *@example
 *   import @arrays
 *   mut nums [int] = [1, 2, 3, 4, 5]
 *   println(arrays.split_every(nums, 2))
 *@end
 */
EzArray ez_arrays_split_every(EzArena *arena, EzArray *arr, int32_t size);

/*@man pair
 *@module arrays
 *@group Transformation
 *@sig pair(a [T], b [T]) -> [[T]]
 *@desc Returns an array of two-element arrays, pairing each element of a with the corresponding element of b. Length is determined by the shorter array.
 *@example
 *   import @arrays
 *   mut keys [int] = [1, 2, 3]
 *   mut vals [int] = [10, 20, 30]
 *   println(arrays.pair(keys, vals))
 *@end
 */
EzArray ez_arrays_pair(EzArena *arena, EzArray *a, EzArray *b);

/* Computation */

/*@man get_sum
 *@module arrays
 *@group Computation
 *@sig get_sum(arr [T]) -> T
 *@desc Returns the sum of all elements. Accepts int, float, or sized numeric types.
 *@example
 *   import @arrays
 *   mut nums [int] = [1, 2, 3, 4]
 *   println(arrays.get_sum(nums))
 *@end
 */
int64_t ez_arrays_get_sum(EzArray *arr);

/*@man get_min
 *@module arrays
 *@group Computation
 *@sig get_min(arr [T]) -> T
 *@desc Returns the smallest element in arr.
 *@example
 *   import @arrays
 *   mut nums [int] = [3, 1, 4, 1, 5]
 *   println(arrays.get_min(nums))
 *@end
 */
int64_t ez_arrays_get_min(EzArray *arr);

/*@man get_max
 *@module arrays
 *@group Computation
 *@sig get_max(arr [T]) -> T
 *@desc Returns the largest element in arr.
 *@example
 *   import @arrays
 *   mut nums [int] = [3, 1, 4, 1, 5]
 *   println(arrays.get_max(nums))
 *@end
 */
int64_t ez_arrays_get_max(EzArray *arr);

/* Sort */

/*@man sort_asc
 *@module arrays
 *@group Modification
 *@sig sort_asc(&arr [T])
 *@desc Sorts arr in ascending order in place. Works on int, float, and string arrays.
 *@example
 *   import @arrays
 *   mut nums [int] = [3, 1, 4, 1, 5]
 *   arrays.sort_asc(nums)
 *   println(nums)
 *@end
 */
void ez_arrays_sort_asc(EzArray *arr);
void ez_arrays_sort_asc_float(EzArray *arr);
void ez_arrays_sort_asc_str(EzArray *arr);

/*@man sort_desc
 *@module arrays
 *@group Modification
 *@sig sort_desc(&arr [T])
 *@desc Sorts arr in descending order in place. Works on int, float, and string arrays.
 *@example
 *   import @arrays
 *   mut nums [int] = [3, 1, 4, 1, 5]
 *   arrays.sort_desc(nums)
 *   println(nums)
 *@end
 */
void ez_arrays_sort_desc(EzArray *arr);
void ez_arrays_sort_desc_float(EzArray *arr);
void ez_arrays_sort_desc_str(EzArray *arr);

#endif
