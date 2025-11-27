package errors

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

// LevenshteinDistance calculates the edit distance between two strings
func LevenshteinDistance(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	// Create matrix
	matrix := make([][]int, len(s1)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(s2)+1)
		matrix[i][0] = i
	}
	for j := range matrix[0] {
		matrix[0][j] = j
	}

	// Fill matrix
	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			cost := 1
			if s1[i-1] == s2[j-1] {
				cost = 0
			}
			matrix[i][j] = min(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len(s1)][len(s2)]
}

func min(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

// EZ language keywords
var Keywords = []string{
	"temp", "const", "do", "return", "if", "or", "otherwise",
	"for", "for_each", "as_long_as", "loop", "break", "continue",
	"in", "not_in", "range", "import", "using", "struct", "enum",
	"nil", "new", "true", "false",
}

// Common builtin functions
var Builtins = []string{
	"len", "typeof", "input", "int", "float", "string", "new",
	"println", "print", "read_int",
}

// SuggestKeyword suggests a keyword if the input is close to one
func SuggestKeyword(input string) string {
	return findClosest(input, Keywords, 2)
}

// SuggestBuiltin suggests a builtin function if the input is close to one
func SuggestBuiltin(input string) string {
	return findClosest(input, Builtins, 2)
}

// SuggestFromList finds the closest match from a list
func SuggestFromList(input string, candidates []string) string {
	return findClosest(input, candidates, 2)
}

// findClosest finds the closest string in a list within maxDistance
func findClosest(input string, candidates []string, maxDistance int) string {
	bestMatch := ""
	bestDistance := maxDistance + 1

	for _, candidate := range candidates {
		dist := LevenshteinDistance(input, candidate)
		if dist < bestDistance && dist <= maxDistance {
			bestDistance = dist
			bestMatch = candidate
		}
	}

	return bestMatch
}
