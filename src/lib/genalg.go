//      genalg.go
//  Autor     : Gerhard Quell - gquell@skequell.de
//  CoAutor   : Claude, Kimi, Zai
//  Copyright : 2026 Gerhard Quell - SKEQuell
//  Erstellt  : 20260626

package lib

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"runtime"
	"sort"
	"strings"
	"sync"
)

type GenType int

const (
	BIT1 GenType = iota
	BIT2
	BIT4
	BIT8
	BITI
	BITF
)

// GenFunc iteriert über einen Genstring und gibt dessen Fitness zurück.
type GenFunc func(gp []byte) float64

// GA verwaltet eine Population paralleler Genstrings.
type GA struct {
	genType    GenType
	genLen     int
	genPar     int
	genFunc    GenFunc
	bitStore   []byte
	intStore   []int
	floatStore []float64
	scores     []float64
	bytesPer   int
}

// ########################################
func parallelStrings(n int, fn func(start, end int)) {
	workers := min(runtime.NumCPU(), n)
	var wg sync.WaitGroup
	chunk := (n + workers - 1) / workers
	for w := range workers {
		start := w * chunk
		end := start + chunk
		if start >= n {
			break
		}
		if end > n {
			end = n
		}
		wg.Add(1)
		go func(s, e int) {
			defer wg.Done()
			fn(s, e)
		}(start, end)
	}
	wg.Wait()
}

// ########################################
func parallelStringsRand(n int, fn func(start, end int, rnd *rand.Rand)) {
	workers := min(runtime.NumCPU(), n)
	seeds := make([]int64, workers)
	for i := range seeds {
		seeds[i] = rand.Int63()
	}
	var wg sync.WaitGroup
	chunk := (n + workers - 1) / workers
	for w := range workers {
		start := w * chunk
		end := start + chunk
		if start >= n {
			break
		}
		if end > n {
			end = n
		}
		rnd := rand.New(rand.NewSource(seeds[w]))
		wg.Add(1)
		go func(s, e int, r *rand.Rand) {
			defer wg.Done()
			fn(s, e, r)
		}(start, end, rnd)
	}
	wg.Wait()
}

// ########################################
func GaCreate(genType GenType, genLen, genPar int, genFunc GenFunc) (*GA, error) {
	// erstellt einen neuen GA-Block
	if genLen <= 0 {
		return nil, errors.New("GaCreate: genLen must be > 0")
	}
	if genPar <= 0 {
		return nil, errors.New("GaCreate: genPar must be > 0")
	}
	if genFunc == nil {
		return nil, errors.New("GaCreate: genFunc is nil")
	}

	ga := &GA{
		genType: genType,
		genLen:  genLen,
		genPar:  genPar,
		genFunc: genFunc,
		scores:  make([]float64, genPar),
	}

	switch genType {
	case BIT1:
		ga.bytesPer = (genLen + 7) / 8
		ga.bitStore = make([]byte, ga.bytesPer*genPar)
	case BIT2:
		ga.bytesPer = (genLen + 3) / 4
		ga.bitStore = make([]byte, ga.bytesPer*genPar)
	case BIT4:
		ga.bytesPer = (genLen + 1) / 2
		ga.bitStore = make([]byte, ga.bytesPer*genPar)
	case BIT8:
		ga.bytesPer = genLen
		ga.bitStore = make([]byte, ga.bytesPer*genPar)
	case BITI:
		ga.intStore = make([]int, genLen*genPar)
	case BITF:
		ga.floatStore = make([]float64, genLen*genPar)
	default:
		return nil, errors.New("GaCreate: unknown GenType")
	}

	return ga, nil
}

// ########################################
func GaInit(ga *GA) error {
	// initialisiert Genstrings mit Zufallswerten
	if ga == nil {
		return errors.New("GaInit: ga is nil")
	}
	switch ga.genType {
	case BIT1, BIT2, BIT4, BIT8:
		parallelStringsRand(len(ga.bitStore), func(start, end int, rnd *rand.Rand) {
			for i := start; i < end; i++ {
				ga.bitStore[i] = byte(rnd.Intn(256))
			}
		})
	case BITI:
		parallelStringsRand(len(ga.intStore), func(start, end int, rnd *rand.Rand) {
			for i := start; i < end; i++ {
				ga.intStore[i] = rnd.Int()
			}
		})
	case BITF:
		parallelStringsRand(len(ga.floatStore), func(start, end int, rnd *rand.Rand) {
			for i := start; i < end; i++ {
				ga.floatStore[i] = rnd.Float64()
			}
		})
	}
	return nil
}

// ########################################
func GaCross(ga *GA, codist int) error {
	// crossover der durch codist bestimmten Abschnitte
	if ga == nil {
		return errors.New("GaCross: ga is nil")
	}
	if codist <= 0 || codist >= ga.genLen {
		return errors.New("GaCross: codist out of range")
	}

	oldPar := ga.genPar
	pairs := oldPar / 2
	newPar := oldPar + 2*pairs

	switch ga.genType {
	case BIT1, BIT2, BIT4, BIT8:
		oldStore := make([]byte, len(ga.bitStore))
		copy(oldStore, ga.bitStore)
		ga.bitStore = append(ga.bitStore, make([]byte, 2*pairs*ga.bytesPer)...)
		ga.genPar = newPar
		ga.scores = make([]float64, newPar)
		parallelStrings(pairs, func(start, end int) {
			for pair := start; pair < end; pair++ {
				i := pair * 2
				p1 := i
				p2 := i + 1
				c1 := oldPar + i
				c2 := oldPar + i + 1
				for gp := 0; gp < ga.genLen; gp++ {
					block := gp / codist
					src := p1
					if block%2 == 1 {
						src = p2
					}
					v, _ := getBitValueRaw(oldStore, ga.bytesPer, ga.genType, src, gp)
					setBitValueRaw(ga.bitStore, ga.bytesPer, ga.genType, c1, gp, v)
					src = p2
					if block%2 == 1 {
						src = p1
					}
					v, _ = getBitValueRaw(oldStore, ga.bytesPer, ga.genType, src, gp)
					setBitValueRaw(ga.bitStore, ga.bytesPer, ga.genType, c2, gp, v)
				}
			}
		})
	case BITI:
		oldStore := make([]int, len(ga.intStore))
		copy(oldStore, ga.intStore)
		ga.intStore = append(ga.intStore, make([]int, 2*pairs*ga.genLen)...)
		ga.genPar = newPar
		ga.scores = make([]float64, newPar)
		parallelStrings(pairs, func(start, end int) {
			for pair := start; pair < end; pair++ {
				i := pair * 2
				p1 := i
				p2 := i + 1
				c1 := oldPar + i
				c2 := oldPar + i + 1
				for gp := 0; gp < ga.genLen; gp++ {
					block := gp / codist
					if block%2 == 0 {
						ga.intStore[c1*ga.genLen+gp] = oldStore[p1*ga.genLen+gp]
						ga.intStore[c2*ga.genLen+gp] = oldStore[p2*ga.genLen+gp]
					} else {
						ga.intStore[c1*ga.genLen+gp] = oldStore[p2*ga.genLen+gp]
						ga.intStore[c2*ga.genLen+gp] = oldStore[p1*ga.genLen+gp]
					}
				}
			}
		})
	case BITF:
		oldStore := make([]float64, len(ga.floatStore))
		copy(oldStore, ga.floatStore)
		ga.floatStore = append(ga.floatStore, make([]float64, 2*pairs*ga.genLen)...)
		ga.genPar = newPar
		ga.scores = make([]float64, newPar)
		parallelStrings(pairs, func(start, end int) {
			for pair := start; pair < end; pair++ {
				i := pair * 2
				p1 := i
				p2 := i + 1
				c1 := oldPar + i
				c2 := oldPar + i + 1
				for gp := 0; gp < ga.genLen; gp++ {
					block := gp / codist
					if block%2 == 0 {
						ga.floatStore[c1*ga.genLen+gp] = oldStore[p1*ga.genLen+gp]
						ga.floatStore[c2*ga.genLen+gp] = oldStore[p2*ga.genLen+gp]
					} else {
						ga.floatStore[c1*ga.genLen+gp] = oldStore[p2*ga.genLen+gp]
						ga.floatStore[c2*ga.genLen+gp] = oldStore[p1*ga.genLen+gp]
					}
				}
			}
		})
	}
	return nil
}

// ########################################
func GaCalc(ga *GA) error {
	// berechnet Werte der ga-Strings parallel und sortiert die Liste
	// genFunc muss thread-sicher sein
	if ga == nil {
		return errors.New("GaCalc: ga is nil")
	}
	if ga.genFunc == nil {
		return errors.New("GaCalc: genFunc is nil")
	}

	type scored struct {
		score float64
		idx   int
	}
	list := make([]scored, ga.genPar)

	parallelStrings(ga.genPar, func(start, end int) {
		for i := start; i < end; i++ {
			gp := extractString(ga, i)
			list[i] = scored{score: ga.genFunc(gp), idx: i}
		}
	})

	sort.Slice(list, func(a, b int) bool {
		return list[a].score > list[b].score
	})

	switch ga.genType {
	case BIT1, BIT2, BIT4, BIT8:
		newStore := make([]byte, len(ga.bitStore))
		for newIdx, s := range list {
			copy(newStore[newIdx*ga.bytesPer:], ga.bitStore[s.idx*ga.bytesPer:(s.idx+1)*ga.bytesPer])
		}
		ga.bitStore = newStore
	case BITI:
		newStore := make([]int, len(ga.intStore))
		for newIdx, s := range list {
			copy(newStore[newIdx*ga.genLen:], ga.intStore[s.idx*ga.genLen:(s.idx+1)*ga.genLen])
		}
		ga.intStore = newStore
	case BITF:
		newStore := make([]float64, len(ga.floatStore))
		for newIdx, s := range list {
			copy(newStore[newIdx*ga.genLen:], ga.floatStore[s.idx*ga.genLen:(s.idx+1)*ga.genLen])
		}
		ga.floatStore = newStore
	}

	ga.scores = make([]float64, ga.genPar)
	for i, s := range list {
		ga.scores[i] = s.score
	}

	return nil
}

// ########################################
func GaSelect(ga *GA, keep int) error {
	// behält die besten keep Strings (vorausgesetzt GaCalc wurde aufgerufen)
	if ga == nil {
		return errors.New("GaSelect: ga is nil")
	}
	if keep <= 0 {
		return errors.New("GaSelect: keep must be > 0")
	}
	if keep > ga.genPar {
		return errors.New("GaSelect: keep larger than genPar")
	}

	switch ga.genType {
	case BIT1, BIT2, BIT4, BIT8:
		ga.bitStore = ga.bitStore[:keep*ga.bytesPer]
	case BITI:
		ga.intStore = ga.intStore[:keep*ga.genLen]
	case BITF:
		ga.floatStore = ga.floatStore[:keep*ga.genLen]
	}
	ga.genPar = keep
	ga.scores = ga.scores[:keep]
	return nil
}

// ########################################
func GaResult(ga *GA) []float64 {
	// gibt eine Liste der ga-Bewertungen zurück
	if ga == nil {
		return nil
	}
	out := make([]float64, len(ga.scores))
	copy(out, ga.scores)
	return out
}

// ########################################
func GaMut(ga *GA, mutf float64) error {
	// mutiert die ga-Strings
	if ga == nil {
		return errors.New("GaMut: ga is nil")
	}
	if mutf < 0 || mutf > 1 {
		return errors.New("GaMut: mutf out of range")
	}

	switch ga.genType {
	case BIT1, BIT2, BIT4, BIT8:
		maxVal := byte(1<<bitsPerGP(ga.genType) - 1)
		parallelStringsRand(ga.genPar, func(start, end int, rnd *rand.Rand) {
			for s := start; s < end; s++ {
				for gp := 0; gp < ga.genLen; gp++ {
					if rnd.Float64() < mutf {
						setBitValueRaw(ga.bitStore, ga.bytesPer, ga.genType, s, gp, byte(rnd.Intn(int(maxVal)+1)))
					}
				}
			}
		})
	case BITI:
		parallelStringsRand(ga.genPar, func(start, end int, rnd *rand.Rand) {
			for s := start; s < end; s++ {
				for gp := 0; gp < ga.genLen; gp++ {
					if rnd.Float64() < mutf {
						ga.intStore[s*ga.genLen+gp] = rnd.Int()
					}
				}
			}
		})
	case BITF:
		parallelStringsRand(ga.genPar, func(start, end int, rnd *rand.Rand) {
			for s := start; s < end; s++ {
				for gp := 0; gp < ga.genLen; gp++ {
					if rnd.Float64() < mutf {
						ga.floatStore[s*ga.genLen+gp] = rnd.Float64()
					}
				}
			}
		})
	}
	return nil
}

// ########################################
func GaPrint(ga *GA, lines int) error {
	// gibt ga-Strings formatiert aus
	if ga == nil {
		return errors.New("GaPrint: ga is nil")
	}
	if lines == 0 {
		return nil
	}
	n := ga.genPar
	if lines > 0 && lines < n {
		n = lines
	}
	var b strings.Builder
	fmt.Fprintf(&b, "idx | score | values\n")
	for i := 0; i < n; i++ {
		score := "-"
		if i < len(ga.scores) {
			score = fmt.Sprintf("%.4f", ga.scores[i])
		}
		switch ga.genType {
		case BIT1, BIT2, BIT4, BIT8:
			fmt.Fprintf(&b, "%3d | %5s |", i, score)
			for gp := 0; gp < ga.genLen; gp++ {
				v, _ := getBitValueRaw(ga.bitStore, ga.bytesPer, ga.genType, i, gp)
				fmt.Fprintf(&b, " %d", v)
			}
			b.WriteString("\n")
		case BITI:
			fmt.Fprintf(&b, "%3d | %5s |", i, score)
			for gp := 0; gp < ga.genLen; gp++ {
				fmt.Fprintf(&b, " %d", ga.intStore[i*ga.genLen+gp])
			}
			b.WriteString("\n")
		case BITF:
			fmt.Fprintf(&b, "%3d | %5s |", i, score)
			for gp := 0; gp < ga.genLen; gp++ {
				fmt.Fprintf(&b, " %.4f", ga.floatStore[i*ga.genLen+gp])
			}
			b.WriteString("\n")
		}
	}
	return WriteOutput(b.String())
}

// ########################################
func bitsPerGP(gt GenType) int {
	switch gt {
	case BIT1:
		return 1
	case BIT2:
		return 2
	case BIT4:
		return 4
	case BIT8:
		return 8
	}
	return 0
}

// ########################################
func getBitValueRaw(store []byte, bytesPer int, gt GenType, sIdx, gpIdx int) (byte, error) {
	switch gt {
	case BIT1:
		byteIdx := sIdx*bytesPer + gpIdx/8
		bitIdx := gpIdx % 8
		return (store[byteIdx] >> bitIdx) & 0x01, nil
	case BIT2:
		byteIdx := sIdx*bytesPer + gpIdx/4
		shift := (gpIdx % 4) * 2
		return (store[byteIdx] >> shift) & 0x03, nil
	case BIT4:
		byteIdx := sIdx*bytesPer + gpIdx/2
		shift := (gpIdx % 2) * 4
		return (store[byteIdx] >> shift) & 0x0F, nil
	case BIT8:
		byteIdx := sIdx*bytesPer + gpIdx
		return store[byteIdx], nil
	}
	return 0, errors.New("getBitValueRaw: unknown GenType")
}

// ########################################
func setBitValueRaw(store []byte, bytesPer int, gt GenType, sIdx, gpIdx int, value byte) error {
	switch gt {
	case BIT1:
		byteIdx := sIdx*bytesPer + gpIdx/8
		bitIdx := gpIdx % 8
		store[byteIdx] = (store[byteIdx] &^ (1 << bitIdx)) | ((value & 0x01) << bitIdx)
		return nil
	case BIT2:
		byteIdx := sIdx*bytesPer + gpIdx/4
		shift := (gpIdx % 4) * 2
		mask := byte(0x03) << shift
		store[byteIdx] = (store[byteIdx] &^ mask) | ((value & 0x03) << shift)
		return nil
	case BIT4:
		byteIdx := sIdx*bytesPer + gpIdx/2
		shift := (gpIdx % 2) * 4
		mask := byte(0x0F) << shift
		store[byteIdx] = (store[byteIdx] &^ mask) | ((value & 0x0F) << shift)
		return nil
	case BIT8:
		byteIdx := sIdx*bytesPer + gpIdx
		store[byteIdx] = value
		return nil
	}
	return errors.New("setBitValueRaw: unknown GenType")
}

// ########################################
func extractString(ga *GA, idx int) []byte {
	switch ga.genType {
	case BIT1, BIT2, BIT4, BIT8:
		start := idx * ga.bytesPer
		end := start + ga.bytesPer
		out := make([]byte, ga.bytesPer)
		copy(out, ga.bitStore[start:end])
		return out
	case BITI:
		start := idx * ga.genLen
		end := start + ga.genLen
		out := make([]byte, ga.genLen*8)
		for i, v := range ga.intStore[start:end] {
			binary.LittleEndian.PutUint64(out[i*8:], uint64(v))
		}
		return out
	case BITF:
		start := idx * ga.genLen
		end := start + ga.genLen
		out := make([]byte, ga.genLen*8)
		for i, v := range ga.floatStore[start:end] {
			binary.LittleEndian.PutUint64(out[i*8:], math.Float64bits(v))
		}
		return out
	}
	return nil
}
