package app

// #include <time.h>
import "C"

const ClocksPerSec = C.CLOCKS_PER_SEC

func Clock() float64 {
	return float64(C.clock())
}
