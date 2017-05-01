load setup_teardown

@test "run command" {
	run_obinex -cmd run -userdir . testbinary.sh
	[ "$status" -eq 0 ]
	sleep 0.5

	diff mock/out/testbinary.sh*/testbinary.sh testbinary.sh
	diff mock/out/testbinary.sh*/output.txt <(echo "$testbin_output")
}

@test "lock command" {
	run_obinex -cmd lock 1h1m1s
	[ "$status" -eq 0 ]
	ls mock/in/lock
	grep "locked" out_watcher

	rm mock/in/lock
	grep "unlocked" out_watcher
}

@test "output command" {
	run_obinex -cmd run testbinary.sh
	[ "$status" == 0 ]
	sleep 3

	run_obinex -cmd output testbinary.sh
	echo "---"
	echo $output
	echo "---"
	[ "$status" == 0 ]
	[ "$output" == "$testbin_output" ]
}

@test "userdir" {
	run_obinex -cmd run testbinary.sh
	[ "$status" == 0 ]
	sleep 1

	run_obinex -cmd output testbinary.sh
	[ "$status" == 0 ]

	ls mock/in/$USER
	ls mock/queued/$USER
	ls mock/executing/$USER
	ls mock/out/$USER
}
