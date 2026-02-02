# Pando: Extremely Scalable BFT Based on Committee Sampling

<img src="etc/logo.png" alt="Logo" width="200"/>

<a href="https://doi.org/10.5281/zenodo.18389201"><img src="https://zenodo.org/badge/DOI/10.5281/zenodo.18389201.svg" alt="DOI"></a>

Byzantine fault-tolerant (BFT) protocols are known to suffer from the scalability issue. Indeed, their performance degrades drastically as the number of replicas $n$ grows. While a long line of work has attempted to achieve the scalability goal, these works can only scale to roughly a hundred replicas, particularly on low-end machines. 

In this paper, we develop BFT protocols from the so-called committee sampling approach that selects a small committee for consensus and conveys the results to all replicas. Such an approach, however, has been focused on the Byzantine agreement (BA) problem (considering replicas only) instead of the BFT problem (in the client-replica model); also, the approach is mainly of theoretical interest only, as concretely, it works for impractically large $n$. 

We build an extremely efficient, scalable, and adaptively secure BFT protocol called Pando in partially synchronous environments based on the committee sampling approach. Our evaluation on Amazon EC2 shows that in contrast to existing protocols, Pando can easily scale to a thousand replicas in the WAN environment, achieving a throughput of 62.57 ktx/sec.  

### Citation
If you are interested in Pando, please find more details in our [paper](https://eprint.iacr.org/2024/664):

```
@misc{cryptoeprint:2024/664,
      author = {Xin Wang and Haochen Wang and Haibin Zhang and Sisi Duan},
      title = {Pando: Extremely Scalable {BFT} Based on Committee Sampling},
      howpublished = {Cryptology {ePrint} Archive, Paper 2024/664},
      year = {2024},
      doi = {10.14722/ndss.2026.230273},
      url = {https://eprint.iacr.org/2024/664}
}
```

## 1. How to run and test #

### 1.1 Dependencies

Our experiments run on Ubuntu 22.04. To compile our code of protocols, we require Go1.23.3 linux/amd64. We also use Python for log summarization. Make sure you have Python 3.10 (or a later version) installed. To prevent testers from downloading dependencies, the codebase we provided includes all necessary dependencies, which are located in ```Pando/src/```. The default configuration file is under ```Pando/etc/conf.json```.


### 1.2 Installation

- Set up the environment variable and disable Go modules mode:

```
cd Pando
go env -w GO111MODULE=on && export GOPATH=$PWD && export GOBIN=$PWD/bin
```

- Compile the code by running the following commands:

```
bash ./scripts/install.sh
```

If no errors are reported after completing the commands above, the compilation is successful. Executable files are expected to appear under ```Pando/bin```.

### 1.3 Execution

#### 1.3.1 Configuration

Modify the configuration file ```Pando/etc/conf.json``` to choose the optional arguments. The [id] of each server should be unique. By default, we use monotonically increasing ids, $0,1,2,...$.

Using the default configuration file, the codebase can be run locally on a single machine. If multiple machines are used for evaluation, please modify the IP addresses and port numbers of all servers manually.

- Generate the key pairs for ECDSA by running the command:

```
bin/ecdsagen [InitID] [EndID]
```

Here, [InitID] is the id of the servers, and [EndID] is the id for generating key pairs between [InitID] and [EndID] (include [EndID]).  

- For example, the following commands generate keys for servers with IDs $0,...,30$ and a client with ID $1000$.

```
    bin/ecdsagen 0 30
    bin/ecdsagen 1000 1000
```

Pando needs at least $7$ servers and $1$ client to run the protocol. For the demo below, we use $30$ servers and $1$ client, so we generate key pairs for server $0,1,..., 30$ and client $1000$ by default. If the keys are successfully generated, they are located under ```Pando/etc/key```".  If the experiments are run locally on one machine, the key generation is already successful. 

If the experiments are run on multiple machines, make sure that the generated keys are placed under the repository of all servers.

- Generate prf key pairs for committee sampling (VRF function):

```
bin/keygen [N] [F]
```

Here, [N] is the total number of replicas in the network, and [F] is the maximum Byzantine failures that the network could tolerate, which equals (N - 1) / 3.  

- For example, the following commands generate prf keys for servers with IDs $0,...,30$.

```
    bin/keygen 31 10
```

If the prf keys are successfully generated, they are located under ```Pando/etc/thresprf_key```".

#### 1.3.2 Demo

We assess evaluation for Pando for $n=31$ by varying the committee size, for example, as $0.8n$.

- Modify the configuration file ```Pando/etc/conf.json```. 

The argument $committeeSizes$ needs to be modified based on the experiment setups. For example, when we evaluate scenario Pando(0.8), we need to modify the $committeeSizes$ to $0.8$ in ```Pando/etc/conf.json```. Also set $batchSize$ as $5000$.

In our code, we provide the configuration with $31$ servers ($id$ from 0 to 30) in ```Pando/etc/conf.json``` on a single local machine, so this preparation step can be skipped. If another configuration aims to be evaluated, please modify the IP address and port numbers of all servers in the configuration file.

- Run the experiment with the command under ```./Pando``` folder:

```
bash ./scripts/runE1.sh
```

This script will first run all $31$ servers in the background (approximately 8 seconds in total). The following outputs are expected to be displayed on the terminal:

```
2025/07/14 09:52:08 **Starting replica 0
2025/07/14 09:52:08 Use ECDSA for authentication
open  /home/starly/Pando/var/log/0/20250714_Normal.log
2025/07/14 09:52:08 [User] User starly
2025/07/14 09:52:08 Leveldb
2025/07/14 09:52:08 User [starly]
2025/07/14 09:52:08 Leveldb
2025/07/14 09:52:08 **Storage option: Data are stored at consensus replicas
2025/07/14 09:52:08 >>>Running Pando protocol!!!
open  /home/starly/Pando/var/log/0/20250714_Eva.log
2025/07/14 09:52:08 Start transmission process...
2025/07/14 09:52:08 Start Pando consensus process...
2025/07/14 09:52:08 ready to listen to port :11000
2025/07/14 09:52:13 ############### epoch 0...
2025/07/14 09:52:14 ############### epoch 1...
2025/07/14 09:52:15 ############### epoch 1...
2025/07/14 09:52:17 ############### epoch 1...
2025/07/14 09:52:18 ############### epoch 1...
2025/07/14 09:52:19 ############### epoch 1...
2025/07/14 09:52:20 ############### epoch 1...
...
```

Since all outputs of $31$ servers will display in one terminal immediately, it is difficult to check each printout manually. Just wait for a few seconds (approximately 15 seconds), the terminal will display "###epoch 1..." every second, which means that all servers are listening to the client request.

Next, the script will send client requests, including 5 transaction blocks, to Pando servers (approximately 60 seconds in total, from epoch 1 to 5). The following outputs are expected:

```
2025/07/14 11:20:52 Starting client test
2025/07/14 11:20:52 ** Client 1000
2025/07/14 11:20:52 User [starly]
create /home/starly/Pando/var/log/1000/
open  /home/starly/Pando/var/log/1000/20250714_Normal.log
2025/07/14 11:20:52 Leveldb
2025/07/14 11:20:52 Use ECDSA for authentication
2025/07/14 11:20:52 User [starly]
2025/07/14 11:20:52 Leveldb
2025/07/14 11:20:52 Starting sec sharing lib
2025/07/14 11:20:52 Starting a write batch, frequency: 1, batchsize: 5000
2025/07/14 11:20:52 ###Client default request len is 1835001
...
```

At the end of the experiment, the output will display:

```
############### epoch 5...
client: no process found
```

After the experiment is successfully launched, evaluation logs can be found in the folder ```./Pando/var/log```. 

- Calculate the scenario performance with the command under ```./Pando``` folder:

```
python3 summarizeE1.py
```

The latency and throughput of scenario Pando(0.8) will be printed, and the result will be recorded in ```Pando/resultE1.txt```.

#### 1.3.3 Note

Due to network bandwidth congestion, the execution could appear to have errors and fail. When failure happens, just restart the experiment before killing all Pando processes:

```
bash ./scripts/killProcess.sh
```
