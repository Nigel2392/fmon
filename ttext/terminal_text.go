package ttext

import (
	"embed"
	"fmt"
	"strings"
	"unicode"
)

const _letters_size = 55

//go:embed letters.txt
var terminal_letters_raw embed.FS
var letters [_letters_size]string

func init() {
	var contents, err = terminal_letters_raw.ReadFile("letters.txt")
	if err != nil {
		panic(fmt.Sprintf("This really shouldn't happen: %v", err))
	}

	var lettersStr = string(contents)
	var split = strings.Split(lettersStr, "\n\n")
	if len(split) != _letters_size {
		panic(fmt.Sprintf(
			"This really shouldn't happen. Invalid letters file: %d != %d",
			len(split), _letters_size,
		))
	}

	letters = [_letters_size]string(split)
}

func fetch(letter string, sub int) string {
	if len(letter) > 1 {
		panic("How many characters does a single ASCII letter have...")
	}

	var asciiNum = int(letter[0]) - sub
	if asciiNum < 0 || asciiNum >= len(letters) {
		panic("index out of range")
	}

	return letters[asciiNum]
}

func Uppercase(letter string) string {
	letter = strings.ToUpper(letter)
	return fetch(letter, 65)
}

func Lowercase(letter string) string {
	letter = strings.ToLower(letter)
	return fetch(letter, (97 - 26))
}

func Space() string {
	return fetch(string(52), 0)
}

func Dot() string {
	return fetch(string(53), 0)
}

func Comma() string {
	return fetch(string(54), 0)
}

func Sentence(s string, processLine ...func(maxHeight int, lineNum int, line string) string) string {
	var letters = make([]string, 0)
	for _, c := range s {
		if !unicode.IsLetter(rune(c)) && c != ' ' && c != '.' && c != ',' {
			panic(fmt.Sprintf("not implemented: %q", string(c)))
		}

		switch {
		case c >= 'A' && c <= 'Z': // uppercase
			letters = append(letters, Uppercase(string(c)))
		case c >= 'a' && c <= 'z': // lowercase
			letters = append(letters, Lowercase(string(c)))
		case c == '.':
			letters = append(letters, Dot())
		case c == ',':
			letters = append(letters, Comma())
		case c == ' ':
			letters = append(letters, Space())
		}
	}

	return Concat(letters, processLine...)
}

func Concat(letters []string, processLine ...func(maxHeight int, lineNum int, line string) string) string {
	var (
		letterM     = make(map[int]int)              // map of letter index -> line length
		letterL     = make([][]string, len(letters)) // 2d array of letter lines
		maxHeight   int
		totalLength int
	)

	for idx, letter := range letters {
		var (
			lines      = strings.Split(letter, "\n")
			lineLength = 0
		)

		if len(lines) > 0 {
			lineLength = len(lines[0])
			totalLength += lineLength
		}

		var currentH = len(lines)
		if currentH > maxHeight {
			maxHeight = currentH
		}

		letterM[idx] = lineLength
		letterL[idx] = lines
	}

	for idx, lineLen := range letterM {
		if len(letterL[idx]) == maxHeight {
			continue
		}

		var (
			add  = strings.Repeat(" ", lineLen)
			addX = maxHeight - len(letterL[idx])
			addS = make([]string, addX, maxHeight)
		)

		for addIdx := range addX {
			addS[addIdx] = add
		}

		letterL[idx] = append(addS, letterL[idx]...)
	}

	var out = new(strings.Builder)
	out.Grow(maxHeight*totalLength + maxHeight)

	var buf = new(strings.Builder)
	for i := 0; i < maxHeight; i++ {
		buf.Reset()
		buf.Grow(totalLength)
		for _, lines := range letterL {
			buf.WriteString(lines[i])
		}

		var ln = buf.String()
		for _, fn := range processLine {
			ln = fn(maxHeight, i, ln)
		}

		out.WriteString(ln)
		out.WriteByte('\n')
	}

	return out.String()
}
