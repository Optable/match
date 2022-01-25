# Benchmarks

The following scatter plot shows the results of benchmarking match attempts using different PSI algorithms on Google Cloud n2-standard-64 [general-purpose virtual machines (VMs)](https://cloud.google.com/compute/docs/general-purpose-machines#n2_machines) (x84-64 architecture). For each benchmark, the sender and the receiver use the same type of VM. The plot shows runtime for various PSI algorithms when the sender and receiver have an equal number of records. The BPSI used for these experiments has a false positive rate fixed at 1e-6. All the match attempts performed have an intersection size of 50m (million). [Detailed benchmarks of the KKRT protocol can be found here](KKRT.md).

<p align="center">
  <img src="scatter_equal_sets.png"/>
</p>

The runtimes for match attempts using different PSI algorithms are provided below. Both sender and receiver used n2-standard-64 VMs with datasets containing 50m, 100m, 200m, 300m, 400m and 500m records. The receiver's datasets are represented row-wise while the sender's datasets are represented column-wise.

<p align="center">
  <img src="heatmap_bpsi.png"/>
</p>

<p align="center">
  <img src="heatmap_npsi.png"/>
</p>

<p align="center">
  <img src="heatmap_kkrt.png"/>
</p>

<p align="center">
  <img src="heatmap_dhpsi.png"/>
</p>