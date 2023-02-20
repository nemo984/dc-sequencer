# dc-sequencer
Total order multicast with sequencer for Distributed Computing class
# Prerequisites
- Go 
- Not Windows (App is using eth0 interface)

# Run
```
git clone https://github.com/nemo984/dc-sequencer
cd dc-sequencer
go mod tidy
./run.sh
```
`run.sh` will spin up 1 seqeuncer program and 2 instances of multicast member that will randomly send messages by default.

Add number to specify the number of member instances.
```
./run.sh 5
```
Above will spin up 5 members.
Press Ctrl-C to stop all. Output will be files in the format of `output-{member_name}.txt`
