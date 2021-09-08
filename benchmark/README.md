# Benchmark

The following graphs show the benchmark results (runtime (in secs)) of running match attempts, using different PSI algorithms, on various Google cloud general-purpose virtual machines (VMs) like n2-standard-32, n2-standard-48, n2-standard-64 and n2-standard-80. Both sender and receiver are using same type of VMs for each experiment. The match attempts are performed between a fixed receiver dataset of size 50m and variety of sender datasets of size 50m, 100m, 150m, 200m and 250m. 

The bpsi used for these experiments have a false positive rate of 1e-6.

![n2-standard-32](n2-standard-32.png)

![n2-standard-48](n2-standard-48.png)

![n2-standard-64](n2-standard-64.png)

![n2-standard-80](n2-standard-80.png)

