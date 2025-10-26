package mathutil

import "errors"

// ErrNegativeNumber возвращается при попытке вычислить факториал отрицательного числа
var ErrNegativeNumber = errors.New("factorial is not defined for negative numbers")

// ErrOverflow возвращается при переполнении результата
var ErrOverflow = errors.New("factorial result overflow")

// Factorial вычисляет факториал числа n.
// Факториал n (обозначается n!) это произведение всех положительных целых чисел от 1 до n.
// По определению: 0! = 1
//
// Параметры:
//   - n: неотрицательное целое число
//
// Возвращает:
//   - uint64: факториал числа n
//   - error: ошибка, если n отрицательное или результат превышает uint64
//
// Примеры:
//   - Factorial(0) = 1
//   - Factorial(5) = 120
//   - Factorial(10) = 3628800
func Factorial(n int) (uint64, error) {
	if n < 0 {
		return 0, ErrNegativeNumber
	}

	// Факториал 0 равен 1 по определению
	if n == 0 || n == 1 {
		return 1, nil
	}

	// Максимальное значение n, для которого факториал помещается в uint64
	// 20! = 2432902008176640000
	// 21! = 51090942171709440000 (переполнение)
	if n > 20 {
		return 0, ErrOverflow
	}

	var result uint64 = 1
	for i := 2; i <= n; i++ {
		result *= uint64(i)
	}

	return result, nil
}

// FactorialRecursive вычисляет факториал числа n рекурсивно.
// Это альтернативная реализация для демонстрации рекурсивного подхода.
// Для больших значений n предпочтительнее использовать итеративную версию (Factorial).
//
// Параметры:
//   - n: неотрицательное целое число
//
// Возвращает:
//   - uint64: факториал числа n
//   - error: ошибка, если n отрицательное или результат превышает uint64
func FactorialRecursive(n int) (uint64, error) {
	if n < 0 {
		return 0, ErrNegativeNumber
	}

	if n > 20 {
		return 0, ErrOverflow
	}

	if n == 0 || n == 1 {
		return 1, nil
	}

	prev, err := FactorialRecursive(n - 1)
	if err != nil {
		return 0, err
	}

	return uint64(n) * prev, nil
}