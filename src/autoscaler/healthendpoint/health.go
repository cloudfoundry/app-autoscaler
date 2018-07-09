package healthendpoint

type Health interface {
	Set(name string, value float64)
	Inc(name string)
	Dec(name string)
}
