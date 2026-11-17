package post

// wordsPerMinute is an average adult reading speed for prose. Technical posts
// read slower, but word_count also counts code and markup, which pads it.
const wordsPerMinute = 230

// ReadingMinutes estimates how long a post with this many words takes to
// read, rounded up, and never less than a minute.
func ReadingMinutes(words int32) int {
	minutes := (int(words) + wordsPerMinute - 1) / wordsPerMinute
	return max(minutes, 1)
}
