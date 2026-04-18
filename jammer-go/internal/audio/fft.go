package audio

import "math"

const fftSize = 512 // must be a power of 2

// fftMagnitudes runs a real-valued radix-2 Cooley-Tukey FFT on the 512
// mono float32 samples in buf (arranged as a ring buffer starting at pos)
// and returns the magnitude of the first 256 bins, scaled to roughly [0,1].
func fftMagnitudes(buf *[fftSize]float32, pos int) []float32 {
	// Copy ring buffer into a complex array in chronological order,
	// applying a Hann window to reduce spectral leakage.
	re := make([]float64, fftSize)
	im := make([]float64, fftSize)
	for i := 0; i < fftSize; i++ {
		sample := float64(buf[(pos+i)&(fftSize-1)])
		window := 0.5 * (1 - math.Cos(2*math.Pi*float64(i)/float64(fftSize-1)))
		re[i] = sample * window
	}

	// In-place iterative Cooley-Tukey FFT (decimation-in-time).
	// Bit-reversal permutation.
	bits := 9 // log2(512)
	for i := 0; i < fftSize; i++ {
		j := bitReverse(i, bits)
		if j > i {
			re[i], re[j] = re[j], re[i]
			im[i], im[j] = im[j], im[i]
		}
	}

	// Butterfly stages.
	for s := 1; s <= bits; s++ {
		m := 1 << s
		half := m >> 1
		wRe := math.Cos(-2 * math.Pi / float64(m))
		wIm := math.Sin(-2 * math.Pi / float64(m))
		for k := 0; k < fftSize; k += m {
			tRe, tIm := 1.0, 0.0
			for j := 0; j < half; j++ {
				uRe, uIm := re[k+j], im[k+j]
				vRe := tRe*re[k+j+half] - tIm*im[k+j+half]
				vIm := tRe*im[k+j+half] + tIm*re[k+j+half]
				re[k+j] = uRe + vRe
				im[k+j] = uIm + vIm
				re[k+j+half] = uRe - vRe
				im[k+j+half] = uIm - vIm
				// Rotate twiddle factor.
				tRe, tIm = tRe*wRe-tIm*wIm, tRe*wIm+tIm*wRe
			}
		}
	}

	// Return magnitude of the first 256 bins, normalised by N/2.
	out := make([]float32, fftSize/2)
	norm := float64(fftSize / 2)
	for i := range out {
		mag := math.Sqrt(re[i]*re[i]+im[i]*im[i]) / norm
		out[i] = float32(mag)
	}
	return out
}

func bitReverse(x, bits int) int {
	r := 0
	for i := 0; i < bits; i++ {
		r = (r << 1) | (x & 1)
		x >>= 1
	}
	return r
}
