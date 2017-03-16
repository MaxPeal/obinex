load setup_teardown

@test "run" {
	run ./obinex -watchdir . -box mock -cmd run testbinary.sh
	sleep 0.5

	diff mock/out/testbinary.sh*/testbinary.sh testbinary.sh
	diff mock/out/testbinary.sh*/output.txt <(echo "executing
executing
executing
Graceful shutdown initiated")
}
