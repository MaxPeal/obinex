load setup_teardown

@test "startup only" {
	grep "serving mock" out_server
	grep "binary requested" out_server

	grep "RPC: localhost connected" out_watcher
	grep "Watcher: watching mock/in" out_watcher
}

@test "execution output" {
	cp testbinary.sh mock/in/foo
	sleep 0.5

	grep "RPC: binary request: mock/executing/foo_.*/foo" out_server
	grep "Server: binary served" out_server

	grep "Watcher: queueing mock/in/foo" out_watcher
}

@test "execution without graceful shutdown" {
	echo "echo foo" > mock/in/foo
	sleep 0.5

	grep "binary request return" out_server
}

@test "execution filecontent" {
	cp testbinary.sh mock/in/foo
	sleep 0.5

	diff mock/out/foo*/foo testbinary.sh
	diff mock/out/foo*/output.txt <(echo "$testbin_output")
}

@test "execution directories" {
	cp testbinary.sh mock/in/foo
	sleep 0.5

	ls mock/queued
	ls mock/executing
	ls mock/out
}

@test "subdirectories" {
	mkdir mock/in/sub

	grep "Watcher: watching mock/in/sub" out_watcher

	cp testbinary.sh mock/in/sub/foo

	ls mock/queued/sub/
	ls mock/executing/sub/
	ls mock/out/sub/
}
