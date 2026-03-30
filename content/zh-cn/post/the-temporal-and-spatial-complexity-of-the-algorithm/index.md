---
pinned: false
tags:
    - 时间复杂度
    - 空间复杂度
categories:
    - 算法
title: "时间与空间复杂度"
description: "算法的复杂度——时间、空间与渐进时间复杂度"
date: 2026-03-09T10:32:00+08:00
image: ""
math: true
license: ""
hidden: false
comments: true
draft: false
---
# 算法复杂度完全指南——时间、空间与渐进时间复杂度

复杂度分析是衡量算法效率的核心工具，帮助我们在编写代码前提前预判程序的性能瓶颈。本文将系统地讲解时间复杂度、空间复杂度，以及 $O$、$\Omega$、$\Theta$ 三种渐进符号的含义与应用，并配合完整样例加以说明。

---

## 什么是复杂度

当我们评估一个算法的好坏时，不能只看它能否得出正确结果，还要看它在**数据量增大时的表现**。复杂度就是用来描述"随输入规模 $n$ 增长，算法所需资源的变化趋势"的数学工具。

- **时间复杂度**：算法需要执行多少步操作？
- **空间复杂度**：算法需要占用多少额外内存？

两者都使用**渐进符号**来表达——忽略常数系数，只关注增长趋势。计算规则如下：

- 只保留最高次项：$3n^2 + 2n + 1 \Rightarrow O(n^2)$
- 忽略常数系数：$5n \Rightarrow O(n)$
- 循环嵌套相乘：两层各跑 $n$ 次 $\Rightarrow O(n^2)$
- 顺序结构取最大：$O(n) + O(n^2) \Rightarrow O(n^2)$

---

## 三种渐进时间复杂度

同一个算法在不同输入下表现可能截然不同。三种渐进时间复杂度分别从**上界、下界、紧确界**三个角度描述算法的行为边界。

### 大 O 符号（上界，最坏情况）

**数学定义**：存在常数 $c > 0$ 和 $n_0$，当 $n \geq n_0$ 时，始终有：

$$f(n) \leq c \cdot g(n)$$

算法的运行时间**最多**是 $g(n)$ 的常数倍，是增长速度的**上限承诺**——保证不会比这更慢。日常开发中使用最广泛，说"这个算法是 $O(n^2)$"通常就是指最坏情况。

```c
// 线性查找 —— O(n)
// 最坏情况：目标在最后，遍历全部 n 个元素
int linear_search(int arr[], int n, int target) {
    for (int i = 0; i < n; i++) {    // 最多执行 n 次
        if (arr[i] == target)
            return i;
    }
    return -1;
}
```

### 大 Ω 符号（下界，最好情况）

**数学定义**：存在常数 $c > 0$ 和 $n_0$，当 $n \geq n_0$ 时，始终有：

$$f(n) \geq c \cdot g(n)$$

算法的运行时间**至少**是 $g(n)$ 的常数倍，是增长速度的**下限承诺**——保证不会比这更快。

```c
// 线性查找 —— Ω(1)
// 最好情况：目标就在第一个位置，只执行 1 次
int linear_search(int arr[], int n, int target) {
    for (int i = 0; i < n; i++) {
        if (arr[i] == target)
            return i;    // 第一次就命中！
    }
    return -1;
}
```

经典结论：任何**基于比较的排序算法**，下界都是 $\Omega(n \log n)$，这是数学可证明的极限，无法突破。

### 大 Θ 符号（紧确界，精确描述）

**数学定义**：存在常数 $c_1, c_2 > 0$ 和 $n_0$，当 $n \geq n_0$ 时，始终有：

$$c_1 \cdot g(n) \leq f(n) \leq c_2 \cdot g(n)$$

算法被 $g(n)$ 从**上下两侧同时夹住**，是最精确的描述。$\Theta$ 成立当且仅当 $O$ 和 $\Omega$ 同时成立且阶数相同。

```c
// 遍历整个数组 —— Θ(n)
// 无论输入如何，都必须访问每个元素，不多不少
int find_max(int arr[], int n) {
    int max_val = arr[0];
    for (int i = 1; i < n; i++) {    // 精确执行 n-1 次，无法提前退出
        if (arr[i] > max_val)
            max_val = arr[i];
    }
    return max_val;
}
```

### 三种符号对比

| 符号 | 含义 | 直觉记忆 | 线性查找举例 |
|------|------|----------|-------------|
| $O(g)$ | 上界 | 最慢不超过这个速度 | $O(n)$，最坏遍历全部 |
| $\Omega(g)$ | 下界 | 最快不低于这个速度 | $\Omega(1)$，最好第一个就找到 |
| $\Theta(g)$ | 紧确界 | 就是这个速度 | 不存在（上下界不同阶） |

线性查找没有 $\Theta$，因为最好与最坏情况的阶数不同，上下界无法合拢。

---

## 时间复杂度

时间复杂度描述算法**执行步骤数**随输入规模 $n$ 的增长趋势。

### 常见阶数对比

| 复杂度 | 名称 | 典型场景 | $n=10^6$ 时的量级 |
|--------|------|----------|-------------------|
| $O(1)$ | 常数时间 | 数组按下标访问、哈希表查找 | 1 次 |
| $O(\log n)$ | 对数时间 | 二分查找、平衡二叉树操作 | ~20 次 |
| $O(n)$ | 线性时间 | 遍历数组、线性查找 | $10^6$ 次 |
| $O(n \log n)$ | 线性对数 | 归并排序、堆排序 | ~$2 \times 10^7$ 次 |
| $O(n^2)$ | 平方时间 | 冒泡排序、选择排序 | $10^{12}$ 次 ⚠️ |
| $O(2^n)$ | 指数时间 | 暴力递归子集枚举 | 不可接受 🚫 |

增长速度：$O(1) < O(\log n) < O(n) < O(n \log n) < O(n^2) < O(2^n)$

### 代码示例

```c
#include <stdio.h>

/* ========== O(1)：常数时间 ========== */
int get_first(int arr[]) {
    return arr[0];    // 与 n 无关，直接返回
}

/* ========== O(log n)：二分查找 ========== */
int binary_search(int arr[], int n, int target) {
    int lo = 0, hi = n - 1;
    while (lo <= hi) {
        int mid = lo + (hi - lo) / 2;
        if (arr[mid] == target)
            return mid;
        else if (arr[mid] < target)
            lo = mid + 1;    // 每次规模减半
        else
            hi = mid - 1;
    }
    return -1;
}

/* ========== O(n)：线性遍历 ========== */
int linear_sum(int arr[], int n) {
    int total = 0;
    for (int i = 0; i < n; i++)    // 执行 n 次
        total += arr[i];
    return total;
}

/* ========== O(n²)：冒泡排序 ========== */
void bubble_sort(int arr[], int n) {
    for (int i = 0; i < n; i++) {
        for (int j = 0; j < n - i - 1; j++) {    // 嵌套循环 → n²
            if (arr[j] > arr[j + 1]) {
                int tmp = arr[j];
                arr[j] = arr[j + 1];
                arr[j + 1] = tmp;
            }
        }
    }
}
```

---

## 空间复杂度

空间复杂度描述算法运行时**额外占用内存**随输入规模的增长趋势（不含输入数据本身）。

### 常见阶数对比

| 复杂度 | 含义 | 典型场景 |
|--------|------|----------|
| $O(1)$ | 固定空间 | 原地排序、用几个临时变量 |
| $O(\log n)$ | 对数空间 | 递归调用栈（二分、快排平均） |
| $O(n)$ | 线性空间 | 复制数组、哈希表、BFS 队列 |
| $O(n^2)$ | 平方空间 | 创建 $n \times n$ 矩阵、邻接矩阵 |

### 代码示例

```c
#include <stdlib.h>

/* ========== O(1)：原地操作 ========== */
int sum_array(int arr[], int n) {
    int total = 0;    // 只有 1 个变量，与 n 无关
    for (int i = 0; i < n; i++)
        total += arr[i];
    return total;
}

/* ========== O(n)：创建新数组 ========== */
int* double_array(int arr[], int n) {
    int* result = malloc(n * sizeof(int));    // 额外占用 n 个空间
    for (int i = 0; i < n; i++)
        result[i] = arr[i] * 2;
    return result;
}

/* ========== O(n)：线性递归调用栈 ========== */
long long factorial(int n) {
    if (n <= 1) return 1;
    return n * factorial(n - 1);    // 递归深度为 n，栈帧占用 O(n)
}

/* ========== O(log n)：对数深度递归 ========== */
int binary_search_rec(int arr[], int target, int lo, int hi) {
    if (lo > hi) return -1;
    int mid = lo + (hi - lo) / 2;
    if (arr[mid] == target)  return mid;
    if (arr[mid] < target)   return binary_search_rec(arr, target, mid + 1, hi);
    else                     return binary_search_rec(arr, target, lo, mid - 1);
    // 递归深度为 log n，栈帧占用 O(log n)
}
```

递归函数每次调用都会在调用栈上分配一个栈帧，**递归深度即为空间复杂度**。深度递归在极端情况下可能导致栈溢出。

---

## 综合样例分析：两数之和

用一道经典问题完整演示三种渐进符号与时空复杂度的分析过程。

## 问题

### 问题描述

给定整数数组 `arr` 和目标值 `target`，找出数组中**和为 `target` 的两个数的下标**。每个输入只有一个答案，不能使用同一元素两次。

### 输入输出

- 输入：`arr = [2, 7, 11, 15]`，`target = 9`
- 输出：`[0, 1]`

### 约束条件

- $2 \leq n \leq 10^4$
- $-10^9 \leq arr[i] \leq 10^9$
- 保证有且只有一个答案

## 思路分析

### 解法一：暴力枚举

枚举所有数对 $(i, j)$，检查是否满足 `arr[i] + arr[j] == target`。思路直接，无需额外空间，但时间效率差。

### 解法二：哈希表优化

遍历数组时，将已见过的值存入哈希表。对每个元素，检查其**补数**（`target - arr[i]`）是否已在表中。若命中则直接返回，否则将当前元素入表。

这是典型的**用空间换时间**：额外花费 $O(n)$ 空间，将时间从 $O(n^2)$ 降至 $O(n)$。

## 代码实现

### 解法一：暴力枚举

```c
// 时间 O(n²)，空间 O(1)
// 返回值：result[0], result[1] 为下标，未找到则返回 {-1, -1}
void two_sum_brute(int arr[], int n, int target, int result[2]) {
    result[0] = result[1] = -1;
    for (int i = 0; i < n; i++) {
        for (int j = i + 1; j < n; j++) {    // 枚举所有数对
            if (arr[i] + arr[j] == target) {
                result[0] = i;
                result[1] = j;
                return;
            }
        }
    }
}
```

### 解法二：哈希表

```c
#include <stdlib.h>
#include <string.h>

// 简单哈希表节点
typedef struct Node {
    int key;      // 数组值
    int val;      // 下标
    struct Node* next;
} Node;

#define TABLE_SIZE 10007

// 时间 O(n)，空间 O(n)
void two_sum_hash(int arr[], int n, int target, int result[2]) {
    Node* table[TABLE_SIZE];
    memset(table, 0, sizeof(table));
    result[0] = result[1] = -1;

    for (int i = 0; i < n; i++) {
        int complement = target - arr[i];    // 所需补数
        // 查表
        int h = ((complement % TABLE_SIZE) + TABLE_SIZE) % TABLE_SIZE;
        for (Node* p = table[h]; p; p = p->next) {
            if (p->key == complement) {
                result[0] = p->val;
                result[1] = i;
                return;
            }
        }
        // 当前元素入表
        int hi = ((arr[i] % TABLE_SIZE) + TABLE_SIZE) % TABLE_SIZE;
        Node* node = malloc(sizeof(Node));
        node->key = arr[i];
        node->val = i;
        node->next = table[hi];
        table[hi] = node;
    }
}
```

## 复杂度与优缺点

### 解法一：暴力枚举

- 时间：$O(n^2)$（最坏），$\Omega(1)$（最好，第一对就命中），无 $\Theta$
- 空间：$\Theta(1)$

- 无需额外空间，内存友好
- 实现简单，无需哈希函数
- 时间效率差，$n = 10^4$ 时已有亿级操作
- 不适合大规模数据

### 解法二：哈希表

- 时间：$\Theta(n)$（必须遍历一次，哈希查找为 $O(1)$）
- 空间：$\Theta(n)$（哈希表最多存 $n$ 个元素）

- 时间效率高，线性扫描一次即可
- 适合大规模数据
- 需要额外 $O(n)$ 内存
- 哈希冲突极端情况下可能退化

### 对比总结

| | 时间（最坏） | 时间（最好） | 时间（$\Theta$） | 空间 | 推荐场景 |
|--|-------------|-------------|-----------------|------|----------|
| 暴力枚举 | $O(n^2)$ | $\Omega(1)$ | — | $O(1)$ | 内存极度受限 |
| 哈希表 | $O(n)$ | $\Omega(n)$ | $\Theta(n)$ | $O(n)$ | 一般业务系统 |

---

## 时间与空间的权衡

在实际工程中，时间和空间往往**不能同时最优**，需要根据场景做出取舍。

- **用空间换时间**（最常见）：哈希表、缓存、动态规划的记忆化数组。
- **用时间换空间**：流式处理大文件时逐行读取，避免一次性加载全部数据到内存。

| 场景 | 推荐策略 |
|------|----------|
| 实时响应、高并发系统 | 牺牲空间，优化时间 |
| 嵌入式设备、内存受限环境 | 牺牲时间，节省空间 |
| 一般业务系统 | 优先优化时间，空间够用即可 |

---

## 常见算法复杂度速查

| 算法 | 时间（$O$） | 时间（$\Omega$） | 时间（$\Theta$） | 空间 |
|------|------------|-----------------|-----------------|------|
| 数组访问 | $O(1)$ | $\Omega(1)$ | $\Theta(1)$ | $O(1)$ |
| 线性查找 | $O(n)$ | $\Omega(1)$ | — | $O(1)$ |
| 二分查找 | $O(\log n)$ | $\Omega(1)$ | — | $O(1)$ |
| 冒泡排序 | $O(n^2)$ | $\Omega(n)$ | $\Theta(n^2)$ | $O(1)$ |
| 归并排序 | $O(n \log n)$ | $\Omega(n \log n)$ | $\Theta(n \log n)$ | $O(n)$ |
| 快速排序 | $O(n^2)$ | $\Omega(n \log n)$ | — | $O(\log n)$ |
| 哈希表查找 | $O(n)$ | $\Omega(1)$ | — | $O(n)$ |

快速排序和线性查找没有 $\Theta$，因为最好与最坏情况的阶数不同，上下界无法合拢。

---