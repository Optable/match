# Benchmarks

The following graphs show the results of benchmarking match attempts using different PSI algorithms on various Google Cloud VM's [general-purpose virtual machines (VMs)](https://cloud.google.com/compute/docs/general-purpose-machines#n2-standard) like n2-standard-64. In each benchmark, the sender and receiver use the same model VM. The receiver has 50m records while the sender has varying datasets of 50m, 100m, 150m, 200m and 250m records. The BPSI used for these experiments has a false positive rate fixed at 1e-6. 

![n2-standard-32](n2-standard-32.png)

![n2-standard-48](n2-standard-48.png)

![n2-standard-64](n2-standard-64.png)

![n2-standard-80](n2-standard-80.png)

The results for match attempts using different PSI algorithms are tabulated below. Both sender and receiver used n2-standard-64 VMS with datasets containing 50m, 100m, 200m, 300m, 400m and 500m records. The receiver's datasets are represented row-wise while the sender's datasets are represented column-wise.

## BPSI

| Time | 50m    | 100m    | 200m     | 300m     | 400m   | 500m     |
|------|-------:|--------:|---------:|--------:|--------:|---------:|
| 50m  | 3m41s | 4m52s | 7m01s    | 9m54s  | 12m23s | 14m47s |
| 100m | 4m19s | 5m34s | 8m07s   | 10m27s | 13m04s  | 15m08s  |
| 200m | 5m21s | 6m36s | 9m22s  | 11m45s | 13m55s | 16m19s |
| 300m | 6m20s | 7m47s | 10m10s  | 12m42s | 14m55s | 17m56s |
| 400m | 7m23s | 9m06s  | 11m28s | 15m16s | 17m08s  | 18m59s |
| 500m | 9m09s  | 9m56s | 12m59s | 15m09s  | 17m31s | 23m04s  |

## NPSI

| Time | 50m      | 100m     | 200m     | 300m   | 400m    | 500m   |
|------|--------:|---------:|--------:|--------:|--------:|--------:|
| 50m  | 3m55s  | 5m06s   | 7m55s  | 11m12s | 13m51s | 17m44s |
| 100m | 4m60s  | 5m58s  | 8m31s  | 11m55s | 14m47s | 18m47s |
| 200m | 7m60s  | 7m48s  | 9m18s  | 12m52s | 15m54s | 19m38s |
| 300m | 11m39s | 11m55s | 13m04s  | 14m32s | 17m24s | 20m30s |
| 400m | 14m25s | 13m55s | 14m37s | 16m17s | 19m44s | 22m59s |
| 500m | 19m43s | 20m14s | 20m45s | 21m17s | 21m13s | 22m57s |

![DHPSI](DHPSI.png)