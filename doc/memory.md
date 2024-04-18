# How is memory calculated ?
Kondense memory resize is an implementation of Meta [Transparent Memory Offloading (TMO)](https://www.cs.cmu.edu/~dskarlat/publications/tmo_asplos22.pdf).

## Why ?
We could think that checking a container memory usage and patching with this value is a good idea but:
1. Libraries used during startup are loaded into memory only to be never touched again afterwards.
2. The Linux filesystem cache doesn't kick out cold data until that memory is required for new data.

Memory usage is not a good proxy for required memory. Instead, **Kondense uses memory pressure** to page out unused memory pages that aren't necessary for nominal workload performance. It dynamically adapts to load peaks and so provides a workingset profile of an application over time.

## Example
An example container has a memory limit of 800M and runs a job in 3m58.050s:
```bash
$ time make -j4 -s
real    3m58.050s
```
However, when we lower the memory limit to 600M, the job finishes in approximately the same amount of time:
```bash
# echo 600M > memory.high

$ time make -j4 -s
real    4m0.654s
```
Clearly, the full 800M aren't required. What if we go even lower than 600M ? Even a 400M limit doesn't materially affect runtime:
```bash
# echo 400M > memory.high

$ time make -j4 -s
real    4m3.186s
```
At 300M, on the other hand, the workload struggles to make forward progress and finish the job in more than 9 minutes:
```bash
# echo 300M > memory.high

$ time make -j4 -s
real    9m9.974s
```
The job of Kondense is to dynamically find the cutoff where job performance begins to plummet.