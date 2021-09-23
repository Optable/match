# Benchmarks

The following graphs show the results of benchmarking match attempts using different PSI algorithms on various Google Cloud VM's [general-purpose virtual machines (VMs)](https://cloud.google.com/compute/docs/general-purpose-machines#n2-standard) like n2-standard-64. In each benchmark, the sender and receiver use the same model VM. The receiver has 50m records while the sender has varying datasets of 50m, 100m, 150m, 200m and 250m records. The BPSI used for these experiments has a false positive rate fixed at 1e-6. 

![n2-standard-32](n2-standard-32.png)

![n2-standard-48](n2-standard-48.png)

![n2-standard-64](n2-standard-64.png)

![n2-standard-80](n2-standard-80.png)

2) Further, we provide a matrix of the running times of performing match attempts using different PSI algorithms on n2-standard-64 VMs between a receiver and a sender with datasets containing 50m, 100m, 200m, 300m, 400m and 500m records. The sizes of the receiver's datasets are represented row wise in each of the matrix, and the sizes of the sender's datasets are represented column wise in each of the matrix.

![BPSI](BPSI.png)

![NPSI](NPSI.png)

![DHPSI](DHPSI.png)