# BatchGCD
Go Library (and program) to perform pairwise gcd on large number of RSA moduli

This implements three different ways to perform pairwise GCD on a large number of RSA moduli.
- Actual pairwise GCD

	This performs n*(n-1)/2 GCD operations on the moduli. This is slow.
	Don't use this.
	
- Accumulating Product

	This iterates over all input moduli, performing a GCD of each one against the product of all previous.
	Once it finds a candidate, it scans all previous moduli to find out which ones it shared a factor with
	(either GCD or division, depending on whether one or both were found).
	The main scan cannot be done in parallel, and even though it seems like this is O(n), the increasing size
	of the accumulated product results it lots of long multiplication and long divison so it's still painfully
	slow for large numbers of moduli.

- Smooth Parts

	DJB's "How to find smooth parts of integers" http://cr.yp.to/papers.html#smoothparts
	This creates a product tree, then converts it to a remainder tree, then kablam you find common factors.
	This is largely the same as the one at https://factorable.net/resources.html
	This is the default, use this unless you're trying to burn in a new CPU or something.
