// Command bench_password_hashing measures the password hashing candidates of
// ADR-0006 on the machine it is run on, and the cost of the parameters that
// record chose.
//
// It exists because a parameter is a decision and not a default. ADR-0006 fixes
// Argon2id at m=64 MiB, t=3, p=2 on numbers taken here; those numbers are a
// property of the hardware, so raising the parameters later means running this
// again rather than guessing again. Nothing else in the project imports it.
//
// It contacts nothing and writes nothing.
//
//	go run ./tools/bench_password_hashing
//	go run ./tools/bench_password_hashing -reps 10 -concurrency 16
package main

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha512"
	"flag"
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/scrypt"
)

// The password every candidate derives from. Length matters to exactly one of
// them: bcrypt reads at most 72 bytes, which is why the check below exists.
const password = "correct horse battery staple"

func main() {
	reps := flag.Int("reps", 5, "how many derivations to time per row; the median is reported")
	concurrency := flag.Int("concurrency", 8, "simultaneous Argon2id derivations for the memory row")
	flag.Parse()

	if err := run(*reps, *concurrency); err != nil {
		fmt.Fprintln(os.Stderr, "bench_password_hashing:", err)
		os.Exit(1)
	}
}

func run(reps, concurrency int) error {
	fmt.Printf("go %s %s/%s, %d CPUs, %d reps per row\n\n",
		runtime.Version(), runtime.GOOS, runtime.GOARCH, runtime.NumCPU(), reps)

	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return err
	}
	pw := []byte(password)

	type row struct {
		name string
		fn   func() error
	}

	rows := []row{
		{"Argon2id  m=19 MiB  t=2  p=1", func() error {
			argon2.IDKey(pw, salt, 2, 19*1024, 1, 32)
			return nil
		}},
		{"Argon2id  m=46 MiB  t=1  p=1", func() error {
			argon2.IDKey(pw, salt, 1, 46*1024, 1, 32)
			return nil
		}},
		{"Argon2id  m=64 MiB  t=2  p=2", func() error {
			argon2.IDKey(pw, salt, 2, 64*1024, 2, 32)
			return nil
		}},
		{"Argon2id  m=64 MiB  t=3  p=2", func() error {
			argon2.IDKey(pw, salt, 3, 64*1024, 2, 32)
			return nil
		}},
		{"Argon2id  m=64 MiB  t=3  p=4", func() error {
			argon2.IDKey(pw, salt, 3, 64*1024, 4, 32)
			return nil
		}},
		{"Argon2id  m=256 MiB t=3  p=2", func() error {
			argon2.IDKey(pw, salt, 3, 256*1024, 2, 32)
			return nil
		}},
		{"scrypt    N=2^15 r=8 p=1 (32 MiB)", func() error {
			_, err := scrypt.Key(pw, salt, 1<<15, 8, 1, 32)
			return err
		}},
		{"scrypt    N=2^16 r=8 p=1 (64 MiB)", func() error {
			_, err := scrypt.Key(pw, salt, 1<<16, 8, 1, 32)
			return err
		}},
		{"bcrypt    cost=10", func() error {
			_, err := bcrypt.GenerateFromPassword(pw, 10)
			return err
		}},
		{"bcrypt    cost=12", func() error {
			_, err := bcrypt.GenerateFromPassword(pw, 12)
			return err
		}},
		{"bcrypt    cost=13", func() error {
			_, err := bcrypt.GenerateFromPassword(pw, 13)
			return err
		}},
		{"PBKDF2-SHA512 210,000 (the reference's)", func() error {
			_, err := pbkdf2.Key(sha512.New, password, salt, 210_000, 64)
			return err
		}},
		{"PBKDF2-SHA512 600,000", func() error {
			_, err := pbkdf2.Key(sha512.New, password, salt, 600_000, 64)
			return err
		}},
		{"PBKDF2-SHA512 2,100,000", func() error {
			_, err := pbkdf2.Key(sha512.New, password, salt, 2_100_000, 64)
			return err
		}},
	}

	fmt.Printf("%-42s %10s\n", "candidate", "median")
	for _, r := range rows {
		d, err := median(reps, r.fn)
		if err != nil {
			return err
		}
		fmt.Printf("%-42s %10s\n", r.name, round(d))
	}

	fmt.Println()
	if err := bcryptTruncation(); err != nil {
		return err
	}

	fmt.Println()
	return concurrent(concurrency, salt, pw)
}

// median times fn reps times and returns the middle duration, which is steadier
// than a mean over a handful of runs on a laptop.
func median(reps int, fn func() error) (time.Duration, error) {
	ds := make([]time.Duration, 0, reps)
	for range reps {
		start := time.Now()
		if err := fn(); err != nil {
			return 0, err
		}
		ds = append(ds, time.Since(start))
	}
	for i := 1; i < len(ds); i++ {
		for j := i; j > 0 && ds[j] < ds[j-1]; j-- {
			ds[j], ds[j-1] = ds[j-1], ds[j]
		}
	}
	return ds[len(ds)/2], nil
}

func round(d time.Duration) time.Duration {
	if d < time.Millisecond {
		return d.Round(10 * time.Microsecond)
	}
	return d.Round(100 * time.Microsecond)
}

// bcryptTruncation reports what this implementation does with a passphrase
// longer than bcrypt's 72-byte input limit: truncate it, so that two different
// passphrases share a hash, or refuse it. It is the one candidate property that
// is a correctness question rather than a cost one.
func bcryptTruncation() error {
	for _, n := range []int{72, 73} {
		pw := make([]byte, n)
		for i := range pw {
			pw[i] = 'a'
		}
		_, err := bcrypt.GenerateFromPassword(pw, bcrypt.MinCost)
		if err != nil {
			fmt.Printf("bcrypt: a %d-byte passphrase is refused: %v\n", n, err)
			continue
		}
		fmt.Printf("bcrypt: a %d-byte passphrase is accepted\n", n)
	}
	return nil
}

// concurrent runs n Argon2id derivations at once and reports wall time and the
// peak live heap sampled while they ran, which is the number that decides
// whether an unauthenticated login route is affordable on a small host.
func concurrent(n int, salt, pw []byte) error {
	runtime.GC()

	done := make(chan struct{})
	peak := make(chan uint64, 1)
	go func() {
		var m runtime.MemStats
		var high uint64
		for {
			select {
			case <-done:
				peak <- high
				return
			default:
			}
			runtime.ReadMemStats(&m)
			if m.HeapAlloc > high {
				high = m.HeapAlloc
			}
			time.Sleep(time.Millisecond)
		}
	}()

	start := time.Now()
	var wg sync.WaitGroup
	for range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			argon2.IDKey(pw, salt, 3, 64*1024, 2, 32)
		}()
	}
	wg.Wait()
	elapsed := time.Since(start)
	close(done)

	fmt.Printf("%d simultaneous Argon2id m=64 MiB t=3 p=2: %s wall, peak live heap %.0f MiB\n",
		n, round(elapsed), float64(<-peak)/(1024*1024))
	return nil
}
