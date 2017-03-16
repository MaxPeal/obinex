load setup_teardown
obinex="./obinex -watchdir . -box mock"

@test "run command" {
	run $obinex -cmd run testbinary.sh
	[ "$status" -eq 0 ]
	sleep 0.5

	diff mock/out/testbinary.sh*/testbinary.sh testbinary.sh
	diff mock/out/testbinary.sh*/output.txt <(echo "executing
executing
executing
Graceful shutdown initiated")
}

@test "lock command" {
	run $obinex -cmd lock 1h1m1s
	[ "$status" -eq 0 ]
	ls mock/in/lock
	grep "locked" out_watcher

	rm mock/in/lock
	grep "unlocked" out_watcher
}
