setup()  {
	./obinex-hwmock 2> out_mock &
	sleep 0.5
	serialpath=$( grep -o '/dev/pts/[0-9]\+' out_mock )
	./obinex-server -watchdir . -serialpath $serialpath -test.coverprofile server_$BATS_TEST_NAME.cov 2> out_server &
	./obinex-watcher -watchdir . -servers localhost -test.coverprofile watcher_$BATS_TEST_NAME.cov 2> out_watcher &
	sleep 2
}

@test "startup only" {
	grep "serving mock" out_server
	grep "binary requested" out_server

	grep "RPC: localhost connected" out_watcher
	grep "Watcher: watching mock/in" out_watcher
}

@test "execution output" {
	echo "somecontent" > mock/in/foo
	sleep 0.5

	grep "RPC: binary request: mock/executing/foo_.*/foo" out_server
	grep "Server: binary served" out_server

	grep "Watcher: queueing mock/in/foo" out_watcher
}

@test "execution filecontent" {
	echo "somecontent" > mock/in/foo
	sleep 0.5

	[ "x$( cat mock/out/foo*/foo )" == "xsomecontent" ]
	diff mock/out/foo*/output.txt <(echo "executing
executing
executing
Graceful shutdown initiated")
}

@test "execution directories" {
	echo "somecontent" > mock/in/foo
	sleep 0.5

	ls mock/queued
	ls mock/executing
	ls mock/out
}

@test "subdirectories" {
	mkdir mock/in/sub

	grep "Watcher: watching mock/in/sub" out_watcher

	echo "somecontent" > mock/in/sub/foo

	ls mock/queued/sub/
	ls mock/executing/sub/
	ls mock/out/sub/
}

teardown() {
	kill $(jobs -p) >/dev/null
	rm -r mock
	rm out_mock out_server out_watcher
}
