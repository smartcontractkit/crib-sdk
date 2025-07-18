package dry

import "fmt"

func ExampleWhen() {
	a := When(true, 1, 2)
	b := When(false, 1, 2)

	fmt.Println(a)
	fmt.Println(b)

	// Output:
	// 1
	// 2
}
