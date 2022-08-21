package toolkit

import "crypto/rand"

//this constant contains all possible characters, that we can use for randomly generated string
//we can use it for example, for creating file names for Linux system.
const randomStringSource = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_+"

// Tools is the type used to instantiate this module.
// Any variables of this type will have access to all methods with reciever *Tools
//T his technic used to share methods from hte modules with other programs
type Tools struct{}

//RandomStringGenerator generates random string of certain length
//it accepts one parameter - lenght of string we want to generate and
//returns the random string
func (t *Tools) RandomStringGenerator(n int) string {

	s := make([]rune, n)
	r := []rune(randomStringSource)

	//Some function details are here
	//rand.Reader is a global, shared instance of a cryptographically secure random number generator

	for i := range s {

		//p returns the number of the given bit length that is prime with high probability.
		//Prime will return error for any error returned by rand
		p, _ := rand.Prime(rand.Reader, len(r))

		x := p.Uint64()
		y := uint64(len(r))

		s[i] = r[x%y]

	}

	return string(s)

}
