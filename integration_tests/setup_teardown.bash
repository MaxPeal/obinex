setup()  {
	./obinex-hwmock 2> out_mock &
	sleep 0.5
	serialpath=$( grep -o '/dev/pts/[0-9]\+' out_mock )
	./obinex-server -watchdir . -serialpath $serialpath -box mock -test.coverprofile server_$BATS_TEST_NAME.cov 2> out_server &
	./obinex-watcher -watchdir . -host localhost -test.coverprofile watcher_$BATS_TEST_NAME.cov 2> out_watcher &
	sleep 2
}

teardown() {
	kill $(jobs -p) >/dev/null
	rm -r mock
	rm out_mock out_server out_watcher
}

run_obinex() {
	run ./obinex -box mock "$@"
}

testbin_output="$(./testbinary.sh; echo 'octopos-shutdown 0')"
