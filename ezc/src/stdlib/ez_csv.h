/*
 * ez_csv.h - @csv module for EZ
 *
 * Copyright (c) 2025-Present Marshall A Burns
 * Licensed under the MIT License. See LICENSE for details.
 */

#ifndef EZ_CSV_H
#define EZ_CSV_H

#include "../runtime/ez_runtime.h"
#include "../runtime/ez_array.h"
#include "ez_io.h" /* EzResult_bool, EzResult_array */

/*@man parse
 *@module csv
 *@group Parsing
 *@sig parse(data string) -> [[string]]
 *@desc Parses a CSV-formatted string and returns an array of rows, where each row is an array of string fields. Also available as csv.decode — both names are identical. Returns a single value; do not use destructuring.
 *@example
 *   import @csv
 *   mut rows = csv.parse("a,b,c\n1,2,3")
 *   for_each row in rows {
 *       println(row)
 *   }
 *@end
 */

/*@man decode
 *@module csv
 *@group Parsing
 *@sig decode(data string) -> [[string]]
 *@desc Alias for csv.parse. Parses a CSV-formatted string and returns an array of rows, where each row is an array of string fields. Returns a single value; do not use destructuring.
 *@example
 *   import @csv
 *   mut rows = csv.decode("name,age\nAlice,30")
 *@end
 */

/*@man encode
 *@module csv
 *@group Formatting
 *@sig encode(rows [[string]]) -> string
 *@desc Converts an array of rows (each row an array of strings) into a CSV-formatted string. Also available as csv.format — both names are identical. Returns a single value; do not use destructuring.
 *@example
 *   import @csv
 *   mut data = {{"Alice", "30"}, {"Bob", "25"}}
 *   mut out = csv.encode(data)
 *   println(out)
 *@end
 */

/*@man format
 *@module csv
 *@group Formatting
 *@sig format(rows [[string]]) -> string
 *@desc Alias for csv.encode. Converts an array of rows (each row an array of strings) into a CSV-formatted string. Returns a single value; do not use destructuring.
 *@example
 *   import @csv
 *   mut data = {{"Alice", "30"}, {"Bob", "25"}}
 *   mut out = csv.format(data)
 *@end
 */

/*@man headers
 *@module csv
 *@group Parsing
 *@sig headers(rows [[string]]) -> [string]
 *@desc Extracts and returns the first row (header row) from parsed CSV data as a flat string array. Returns a single value; do not use destructuring.
 *@example
 *   import @csv
 *   mut rows = csv.parse("name,age\nAlice,30")
 *   mut h = csv.headers(rows)
 *   println(h)
 *@end
 */

/*@man read_file
 *@module csv
 *@group File I/O
 *@sig read_file(path string) -> ([[string]], Error)
 *@desc Reads the CSV file at path and returns parsed rows as an array of string arrays. Always use destructuring — single-variable assignment is a compile error. Relative paths resolve from the working directory where the binary is executed, not the source file location.
 *@example
 *   import @csv
 *   mut rows, err = csv.read_file("data.csv")
 *   if err != nil { println("failed: ${err}") }
 *   mut rows, _ = csv.read_file("data.csv")
 *@end
 */

/*@man write_file
 *@module csv
 *@group File I/O
 *@sig write_file(path string, rows [[string]]) -> (bool, Error)
 *@desc Writes an array of rows (each row an array of strings) to a CSV file at path. Creates the file if it does not exist; overwrites if it does. Always use destructuring — single-variable assignment is a compile error. Relative paths resolve from the working directory where the binary is executed, not the source file location.
 *@example
 *   import @csv
 *   mut data = {{"name", "age"}, {"Alice", "30"}}
 *   mut ok, err = csv.write_file("out.csv", data)
 *   if err != nil { println("failed: ${err}") }
 *@end
 */

/* Parse CSV string into array of arrays of strings */
EzArray ez_csv_parse(EzArena *arena, EzString csv_string);

/* Convert array of arrays of strings to CSV string */
EzString ez_csv_stringify(EzArena *arena, EzArray *data);

/* Extract the first row (headers) from parsed CSV data */
EzArray ez_csv_headers(EzArena *arena, EzArray *data);

/* Read CSV file */
EzArray ez_csv_read(EzArena *arena, EzString path);

/* Write CSV to file */
bool ez_csv_write(EzArena *arena, EzString path, EzArray *data);

/* _result variants */
EzResult_array ez_csv_read_result(EzArena *arena, EzString path);
EzResult_bool ez_csv_write_result(EzArena *arena, EzString path, EzArray *data);

#endif
