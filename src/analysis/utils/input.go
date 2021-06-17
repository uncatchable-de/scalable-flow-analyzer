package utils

import (
	"fmt"
	"log"
	"strconv"
	"strings"
)

// Ask User for Confirmation
// From https://gist.github.com/albrow/5882501
func AskForConfirmation(message string) bool {
	fmt.Println(message)
	var response string
	_, err := fmt.Scanln(&response)
	if err != nil {
		log.Fatal(err)
	}
	okayResponses := []string{"y", "Y", "yes", "Yes", "YES"}
	nokayResponses := []string{"n", "N", "no", "No", "NO"}
	switch {
	case containsString(okayResponses, response):
		return true
	case containsString(nokayResponses, response):
		return false
	default:
		fmt.Println("Please type yes or no and then press enter:")
		return AskForConfirmation(message)
	}
}

// posString returns the first index of element in slice.
// If slice does not contain element, returns -1.
func posString(slice []string, element string) int {
	for index, elem := range slice {
		if elem == element {
			return index
		}
	}
	return -1
}

// containsString returns true iff slice contains element
func containsString(slice []string, element string) bool {
	return posString(slice, element) != -1
}

// ExpandIntegerList returns from a string all integers in this range
// e.g. 2-5,12 will return 2,3,4,5,12
func ExpandIntegerList(str string) []uint16 {
	ints := make([]uint16, 0)
	for _, intsSeperated := range strings.Split(str, ",") {
		if intsSeperated == "" {
			continue
		}
		if strings.Contains(intsSeperated, "-") {
			startEnd := strings.SplitN(intsSeperated, "-", 2)
			start, err := strconv.ParseUint(startEnd[0], 10, 16)
			if err != nil {
				panic("Wrong argument to getListOfInts: " + intsSeperated)
			}
			end, err := strconv.ParseUint(startEnd[1], 10, 16)
			if err != nil {
				panic("Wrong argument to getListOfInts: " + intsSeperated)
			}
			for i := start; i <= end; i++ {
				ints = append(ints, uint16(i))
			}
		} else {
			integer, err := strconv.ParseUint(intsSeperated, 10, 16)
			if err != nil {
				panic("Wrong argument to getListOfInts: " + intsSeperated)
			}
			ints = append(ints, uint16(integer))
		}
	}
	return ints
}
