package main

func testFunction() int {
	x := 42
	y := x * 2
	return y
}

func main() {
	result := testFunction()
	println("Result:", result)
}
