package model

import (
	"testing"
)

func Test_findAtToken(t *testing.T) {
	tests := []struct {
		name      string
		runes     string
		position  int
		wantWord  string
		wantIndex int
	}{
		{
			name:      "position_0_nothing_to_left",
			runes:     "@actor",
			position:  0,
			wantWord:  "",
			wantIndex: -1,
		},
		{
			name:      "cursor_after_actor_returns_actor_and_index_of_@",
			runes:     "@actor",
			position:  6,
			wantWord:  "actor",
			wantIndex: 0,
		},
		{
			name:      "cursor_after_actor_in_run_actor_returns_actor_and_index_of_@",
			runes:     "run @actor",
			position:  10,
			wantWord:  "actor",
			wantIndex: 4,
		},
		{
			name:      "word_with_no_at_before_it_returns_not_found",
			runes:     "actor",
			position:  5,
			wantWord:  "",
			wantIndex: -1,
		},
		{
			name:      "bare_at_with_no_word_returns_not_found",
			runes:     "@",
			position:  1,
			wantWord:  "",
			wantIndex: -1,
		},
		{
			name:      "word_can_contain_underscore",
			runes:     "@actor_1",
			position:  8,
			wantWord:  "actor_1",
			wantIndex: 0,
		},
		{
			name:      "cursor_right_after_at_with_space_before_no_word_runes_before_at_returns_not_found",
			runes:     " @actor",
			position:  2,
			wantWord:  "",
			wantIndex: -1,
		},
		{
			name:      "cursor_after_space_following_actor_does_not_match",
			runes:     "@actor ",
			position:  7,
			wantWord:  "",
			wantIndex: -1,
		},
		{
			name:      "single_at_x_returns_x",
			runes:     "@x",
			position:  2,
			wantWord:  "x",
			wantIndex: 0,
		},
		{
			name:      "cursor_in_middle_of_word_finds_token_from_at_to_cursor",
			runes:     "@actor",
			position:  3,
			wantWord:  "ac",
			wantIndex: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotWord, gotIndex := findAtToken([]rune(tt.runes), tt.position)
			if gotWord != tt.wantWord {
				t.Errorf("findAtToken() word = %q, want %q", gotWord, tt.wantWord)
			}
			if gotIndex != tt.wantIndex {
				t.Errorf("findAtToken() index = %d, want %d", gotIndex, tt.wantIndex)
			}
		})
	}
}

func Test_flatOffset(t *testing.T) {
	tests := []struct {
		name   string
		runes  string
		row    int
		offset int
		want   int
	}{
		{
			name:   "empty_runes_returns_zero",
			runes:  "",
			row:    0,
			offset: 0,
			want:   0,
		},
		{
			name:   "line_0_offset_0_returns_start_of_line",
			runes:  "ab\ncde",
			row:    0,
			offset: 0,
			want:   0,
		},
		{
			name:   "line_0_offset_1_returns_index_1",
			runes:  "ab\ncde",
			row:    0,
			offset: 1,
			want:   1,
		},
		{
			name:   "line_1_offset_0_returns_start_of_second_line",
			runes:  "ab\ncde",
			row:    1,
			offset: 0,
			want:   3,
		},
		{
			name:   "line_1_offset_2_returns_index_5",
			runes:  "ab\ncde",
			row:    1,
			offset: 2,
			want:   5,
		},
		{
			name:   "row_past_last_line_returns_len_runes",
			runes:  "ab\ncde",
			row:    2,
			offset: 0,
			want:   6,
		},
		{
			name:   "single_line_offset_2",
			runes:  "abc",
			row:    0,
			offset: 2,
			want:   2,
		},
		{
			name:   "three_lines_middle_line_offset_1",
			runes:  "a\nbc\ndef",
			row:    1,
			offset: 1,
			want:   3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := flatOffset([]rune(tt.runes), tt.row, tt.offset)
			if got != tt.want {
				t.Errorf("flatOffset(%q, row=%d, offset=%d) = %d, want %d", tt.runes, tt.row, tt.offset, got, tt.want)
			}
		})
	}
}

func Test_isWordRune(t *testing.T) {
	tests := []struct {
		name string
		r    rune
		want bool
	}{
		{
			name: "letter_lowercase",
			r:    'a',
			want: true,
		},
		{
			name: "letter_uppercase",
			r:    'Z',
			want: true,
		},
		{
			name: "digit_zero",
			r:    '0',
			want: true,
		},
		{
			name: "digit_nine",
			r:    '9',
			want: true,
		},
		{
			name: "underscore",
			r:    '_',
			want: true,
		},
		{
			name: "space",
			r:    ' ',
			want: false,
		},
		{
			name: "at_sign",
			r:    '@',
			want: false,
		},
		{
			name: "newline",
			r:    '\n',
			want: false,
		},
		{
			name: "hyphen",
			r:    '-',
			want: false,
		},
		{
			name: "dot",
			r:    '.',
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isWordRune(tt.r); got != tt.want {
				t.Errorf("isWordRune(%q) = %v, want %v", tt.r, got, tt.want)
			}
		})
	}
}

func Test_lookupSavedQuery(t *testing.T) {
	queries := []SavedQuery{
		{Name: "actor", Query: "SELECT * FROM actors"},
		{Name: "film", Query: "SELECT * FROM film"},
		{Name: "address", Query: "SELECT * FROM address"},
	}
	tests := []struct {
		name       string
		queries    []SavedQuery
		searchName string
		wantQuery  string
		wantFound  bool
	}{
		{
			name:       "exact_match_returns_query",
			queries:    queries,
			searchName: "actor",
			wantQuery:  "SELECT * FROM actors",
			wantFound:  true,
		},
		{
			name:       "case_insensitive_match",
			queries:    queries,
			searchName: "ACTOR",
			wantQuery:  "SELECT * FROM actors",
			wantFound:  true,
		},
		{
			name:       "no_match_returns_not_found",
			queries:    queries,
			searchName: "missing",
			wantQuery:  "",
			wantFound:  false,
		},
		{
			name:       "empty_list_returns_not_found",
			queries:    nil,
			searchName: "actor",
			wantQuery:  "",
			wantFound:  false,
		},
		{
			name:       "first_match_wins_when_duplicate_names",
			queries:    []SavedQuery{{Name: "x", Query: "first"}, {Name: "x", Query: "second"}},
			searchName: "x",
			wantQuery:  "first",
			wantFound:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotQuery, gotFound := lookupSavedQuery(tt.queries, tt.searchName)
			if gotQuery != tt.wantQuery {
				t.Errorf("lookupSavedQuery() query = %q, want %q", gotQuery, tt.wantQuery)
			}
			if gotFound != tt.wantFound {
				t.Errorf("lookupSavedQuery() found = %v, want %v", gotFound, tt.wantFound)
			}
		})
	}
}
