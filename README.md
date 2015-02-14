# BatchGCD
Go Library (and program) to perform pairwise gcd on large number of RSA moduli

Note that the code at https://factorable.net/resources.html is way faster than this, mostly due to it's use of
the GMP library.  This might change at some point, if the golang math/big library improves.

This implements three different ways to perform pairwise GCD on a large number of RSA moduli.
- Actual pairwise GCD

	This performs n*(n-1)/2 GCD operations on the moduli. This is slow.
	Don't use this
- Accumulating Product

	This iterates over all input moduli, performing a GCD of each one against the product of all previous.
	It runs in O(n) time for finding candidates, but then each candidate needs to scan all previous moduli to find out
	which one it shared a factor with (either GCD or division).
	This main scan cannot be done in parallel at all
	Use this if memory usage is more of a concern than time
- Smooth Parts

	DJB's "How to find smooth parts of integers" http://cr.yp.to/papers.html#smoothparts
	This creates a product tree, then converts it to a remainder tree, then kablam you find common factors.
	Pretty awesome, but uses more memory (n*log2(n) -- but without a lot of the overhead.
	Again, use the one at https://factorable.net/resources.html if you can, it's way faster.
	This is the default
