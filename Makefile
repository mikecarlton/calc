
rcalc: rcalc.rs
	rustc rcalc.rs

install: rcalc
	mv calc ~/bin/rcalc

