package hyperloglog

import (
	"bufio"
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"math"
	"math/rand"
	"os"
	"testing"
	"time"
)

// Return a dictionary up to n words. If n is zero, return the entire
// dictionary.
func dictionary(n int) []string {
	var words []string
	dict := "/usr/share/dict/words"
	f, err := os.Open(dict)
	if err != nil {
		fmt.Printf("can't open dictionary file '%s': %v\n", dict, err)
		os.Exit(1)
	}
	count := 0
	buf := bufio.NewReader(f)
	for {
		if n != 0 && count >= n {
			break
		}
		word, err := buf.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			continue
		}
		words = append(words, word)
		count++
	}
	f.Close()
	return words
}

func geterror(actual uint64, estimate uint64) (result float64) {
	return (float64(estimate) - float64(actual)) / float64(actual)
}

func testHyperLogLog(t *testing.T, n, low_b, high_b int) {
	words := dictionary(n)
	bad := 0
	n_words := uint64(len(words))
	for i := low_b; i < high_b; i++ {
		m := uint(math.Pow(2, float64(i)))

		h, err := New(m)
		if err != nil {
			t.Fatalf("can't make New(%d): %v", m, err)
		}

		hash := fnv.New32()
		for _, word := range words {
			hash.Write([]byte(word))
			h.Add(hash.Sum32())
			hash.Reset()
		}

		expected_error := 1.04 / math.Sqrt(float64(m))
		actual_error := math.Abs(geterror(n_words, h.Count()))

		if actual_error > expected_error {
			bad++
			t.Logf("m=%d: error=%.5f, expected <%.5f; actual=%d, estimated=%d\n",
				m, actual_error, expected_error, n_words, h.Count())
		}

	}
	t.Logf("%d of %d tests exceeded estimated error", bad, high_b-low_b)
}

func TestHyperLogLogSmall(t *testing.T) {
	testHyperLogLog(t, 5, 4, 17)
}

func TestHyperLogLogBig(t *testing.T) {
	testHyperLogLog(t, 0, 4, 17)
}

var src = rand.NewSource(time.Now().UnixNano())

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

func randStringBytesMaskImprSrc(n int) string {
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}

func TestHyperLogLogIntersectLarge(t *testing.T) {
	rand.Seed(time.Now().UnixNano())

	runs := uint64(10000000)
	registerSize := uint(2048)

	a, _ := New(registerSize)
	b, _ := New(registerSize)
	hash := fnv.New32()

	for i := uint64(0); i < runs; i++ {
		hash.Write([]byte(randStringBytesMaskImprSrc(10)))
		s := hash.Sum32()
		a.Add(s)
		if i%2 == 0 {
			b.Add(s)
		}
		hash.Reset()
	}

	intersected, _ := a.Intersect(b)
	maxIntersect := (float64(runs) / 2) * 1.022
	minIntersect := (float64(runs) / 2) * 0.988

	if float64(intersected) > maxIntersect || float64(intersected) < minIntersect {
		log.Printf("Intersect exceeded deviation bounderies (min: %f max: %f result: %d)", minIntersect, maxIntersect, intersected)
		t.Fail()
	}
}

func TestHyperLogLogIntersectSmall(t *testing.T) {
	a, _ := New(2048)
	b, _ := New(2048)

	hash := fnv.New32()

	// Apple in both
	hash.Write([]byte("apple"))
	s := hash.Sum32()
	a.Add(s)
	b.Add(s)
	hash.Reset()

	// Beer in both
	hash.Write([]byte("beer"))
	s = hash.Sum32()
	a.Add(s)
	b.Add(s)
	hash.Reset()

	// Banana in a
	hash.Write([]byte("banana"))
	s = hash.Sum32()
	a.Add(s)
	hash.Reset()

	// Pineapple in a
	hash.Write([]byte("pineapple"))
	s = hash.Sum32()
	b.Add(s)
	hash.Reset()

	intersected, _ := a.Intersect(b)
	log.Printf("intersect %d", intersected)
}

func TestHyperLogLogIntersectNone(t *testing.T) {
	a, _ := New(2048)
	b, _ := New(2048)

	hash := fnv.New32()

	// Apple in both
	hash.Write([]byte("apple"))
	s := hash.Sum32()
	a.Add(s)
	hash.Reset()

	// Beer in both
	hash.Write([]byte("beer"))
	s = hash.Sum32()
	a.Add(s)
	hash.Reset()

	// Banana in a
	hash.Write([]byte("banana"))
	s = hash.Sum32()
	a.Add(s)
	hash.Reset()

	intersected, _ := a.Intersect(b)
	log.Printf("none %d", intersected)
}

func benchmarkCount(b *testing.B, registers int) {
	words := dictionary(0)
	m := uint(math.Pow(2, float64(registers)))

	h, err := New(m)
	if err != nil {
		return
	}

	hash := fnv.New32()
	for _, word := range words {
		hash.Write([]byte(word))
		h.Add(hash.Sum32())
		hash.Reset()
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		h.Count()
	}
}

func BenchmarkCount4(b *testing.B) {
	benchmarkCount(b, 4)
}

func BenchmarkCount5(b *testing.B) {
	benchmarkCount(b, 5)
}

func BenchmarkCount6(b *testing.B) {
	benchmarkCount(b, 6)
}

func BenchmarkCount7(b *testing.B) {
	benchmarkCount(b, 7)
}

func BenchmarkCount8(b *testing.B) {
	benchmarkCount(b, 8)
}

func BenchmarkCount9(b *testing.B) {
	benchmarkCount(b, 9)
}

func BenchmarkCount10(b *testing.B) {
	benchmarkCount(b, 10)
}
