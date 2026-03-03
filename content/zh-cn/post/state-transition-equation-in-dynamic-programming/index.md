---
title: "状态转移方程，与动态规划"
date: 2026-03-02T13:57:00+08:00
description: "对动态规划与状态转移方程的总结，及常见模型"
tags:
    - 动态规划
    - 状态转移方程
categories:
    - 算法
draft: false
math: true
comments: true
hidden: false
pinned: false
---

在算法题当中，我们经常可以看见动态规划的影子，所以在此总结一下动态规划（DP）和其中非常重要的一个部分——状态转移方程。

## 一、何为动态规划

动态规划（Dynamic Programming，DP）是一种通过把原问题分解为子问题来求解的算法思想。

动态规划不是某种具体的数据结构，而是一种思维方式。

DP 需要满足以下两个性质：

### 1. 最优子结构

原问题的最优解包含子问题的最优解。

### 2. 重叠子问题

子问题被反复计算，可以缓存避免重复。

## 二、何为状态转移方程

要了解状态转移方程，首先应当知道何为"状态"。

### 1. "状态"

状态是对问题在某一阶段的描述，通常用 `dp[i]` 或 `dp[i][j]` 表示。

例如：
- `dp[i]` = 前 i 个元素的最优解
- `dp[i][j]` = 从位置 i 到位置 j 的最优解
- `dp[i][w]` = 考虑前 i 个物品、剩余容量为 w 时的最优解

### 2. 状态转移方程

状态转移方程可以粗略的写成：

```
新状态 = f(旧状态)
```

状态转移方程所做的是确定该步有什么选择，以及选择背后所对应的子问题（可以理解为递推式）。

## 三、典型例题

## 三、典型例题

### 例 1：线性 DP - 爬楼梯

#### 问题

每次可以爬 1 或 2 个台阶，爬到第 n 阶有多少种方法？

#### 思路分析

1. 定义状态：`dp[i]` = 爬到第 i 阶的方法数
2. 最后一步分析：到达第 i 阶，只能从第 i-1 阶（迈 1 步）或第 i-2 阶（迈 2 步）过来
3. 根据分析，我们可以得到这样一个状态转移方程：

$$dp[i] = dp[i-1] + dp[i-2]$$

注意：此状态转移方程还有两个边界状态：`dp[1] = 1`，`dp[2] = 2`

4. 图解：

```
i = 5
dp[1] = 1
dp[2] = 2
dp[3] = 3
dp[4] = 5
dp[5] = 8
```

#### 代码实现

```c
#include <stdio.h>

int climbStairs(int n) {
    if (n <= 2) return n;
    int dp[n + 1];
    dp[1] = 1;
    dp[2] = 2;
    for (int i = 3; i <= n; i++) {
        dp[i] = dp[i-1] + dp[i-2];  // 状态转移方程
    }
    return dp[n];
}

int main() {
    printf("爬 5 阶楼梯的方法数: %d\n", climbStairs(5));  // 输出: 8
    return 0;
}
```

#### 时间复杂度与优缺点

**时间复杂度** ：$O(n)$  
**空间复杂度** ：$O(n)$

**优点** ：
- ✅ 问题简单，易于理解
- ✅ 状态定义直观

**缺点** ：
- ❌ 只能计算一固定地方的值，若要计算多次则需多次递推

---

### 例 2：线性 DP - 打家劫舍

#### 问题

一排房子不能抢相邻的，求最大金额。给定数组 `nums` 表示各房子的金额。

#### 思路分析

1. 定义状态：`dp[i]` = 抢到第 i 间房能获得的最大金额
2. 最后一步分析：第 i 间房，要么抢，要么不抢
   - 抢这间：得到 `nums[i] + dp[i-2]`（前面最多抢到 i-2 间）
   - 不抢这间：得到 `dp[i-1]`（抢到 i-1 间的最大值）
3. 状态转移方程：

$$dp[i] = \max(dp[i-1], dp[i-2] + nums[i])$$

4. 图解：

```
nums = [2, 7, 9, 3, 1]
dp[0] = 2
dp[1] = max(2, 7) = 7
dp[2] = max(7, 2+9) = 11
dp[3] = max(11, 7+3) = 11
dp[4] = max(11, 11+1) = 12  (抢第 0、2、4 间：2+9+1=12)
```

#### 代码实现

```c
#include <stdio.h>

int max(int a, int b) { return a > b ? a : b; }

int rob(int *nums, int n) {
    if (n == 1) return nums[0];
    int dp[n];
    dp[0] = nums[0];
    dp[1] = max(nums[0], nums[1]);
    for (int i = 2; i < n; i++) {
        dp[i] = max(dp[i-1], dp[i-2] + nums[i]);  // 状态转移方程
    }
    return dp[n-1];
}

int main() {
    int nums[] = {2, 7, 9, 3, 1};
    int n = sizeof(nums) / sizeof(nums[0]);
    printf("最大金额: %d\n", rob(nums, n));  // 输出: 12
    return 0;
}
```

#### 时间复杂度与优缺点

**时间复杂度** ：$O(n)$  
**空间复杂度** ：$O(n)$，可优化为 $O(1)$（仅保留前两项）

**优点** ：
- ✅ 与爬楼梯类似，思路清晰
- ✅ 可优化空间至 O(1)

**缺点** ：
- ❌ 不能直接回溯具体哪些房子被抢

---

### 例 3：背包 DP - 0/1 背包

#### 问题

n 个物品，重量 `w[]`，价值 `v[]`，背包容量 W，求最大价值。

#### 思路分析

1. 定义状态：`dp[i][j]` = 考虑前 i 个物品、容量为 j 时的最大价值
2. 最后一步分析：第 i 个物品，放或不放
   - 不放：`dp[i][j] = dp[i-1][j]`
   - 放：`dp[i][j] = dp[i-1][j - w[i]] + v[i]`（需 $j \geq w[i]$）
3. 状态转移方程：

$$dp[i][j] = \max(dp[i-1][j], dp[i-1][j - w[i]] + v[i])$$

4. 图解：

物品：(w=2, v=3)、(w=3, v=4)、(w=4, v=5)   背包容量 W=5

```
      j=0  j=1  j=2  j=3  j=4  j=5
i=0    0    0    0    0    0    0
i=1    0    0    3    3    3    3
i=2    0    0    3    4    4    7   ← 放物品 1 + 物品 2，价值=3+4=7
i=3    0    0    3    4    5    7

答案：dp[3][5] = 7
```

#### 代码实现

```c
#include <stdio.h>

#define MAX_N 105
#define MAX_W 1005

int dp[MAX_N][MAX_W];

int max(int a, int b) { return a > b ? a : b; }

int knapsack(int *w, int *v, int n, int W) {
    for (int i = 0; i <= n; i++)
        for (int j = 0; j <= W; j++)
            dp[i][j] = 0;

    for (int i = 1; i <= n; i++) {
        for (int j = 0; j <= W; j++) {
            dp[i][j] = dp[i-1][j];                          // 不放
            if (j >= w[i-1])                                 // 放得下
                dp[i][j] = max(dp[i][j],
                               dp[i-1][j - w[i-1]] + v[i-1]);  // 状态转移方程
        }
    }
    return dp[n][W];
}

int main() {
    int w[] = {2, 3, 4};
    int v[] = {3, 4, 5};
    int n = 3, W = 5;
    printf("最大价值: %d\n", knapsack(w, v, n, W));  // 输出: 7
    return 0;
}
```

**空间优化** ：二维 dp 可以压缩成一维，内层循环**必须倒序**，防止同一件物品被重复放入：

```c
for (int i = 0; i < n; i++)
    for (int j = W; j >= w[i]; j--)        // 倒序！
        dp[j] = max(dp[j], dp[j - w[i]] + v[i]);
```

#### 时间复杂度与优缺点

**时间复杂度** ：$O(nW)$  
**空间复杂度** ：$O(nW)$，优化后为 $O(W)$

**优点** ：
- ✅ DP 框架经典，易于扩展
- ✅ 可一次性解决所有容量的最优值

**缺点** ：
- ❌ 物品数量或容量很大时，时间空间压力大

---

### 例 4：序列 DP - 最长公共子序列（LCS）

#### 问题

两个字符串，求最长公共子序列的长度。例：`"abcde"` 和 `"ace"` → 长度为 3（`ace`）

#### 思路分析

1. 定义状态：`dp[i][j]` = s1 前 i 个字符与 s2 前 j 个字符的 LCS 长度
2. 最后一步分析：s1[i] 和 s2[j] 是否相等：
   - 相等：`dp[i][j] = dp[i-1][j-1] + 1`
   - 不相等：`dp[i][j] = max(dp[i-1][j], dp[i][j-1])`（去掉 s1[i] 或 s2[j]，看哪个更优）

3. 状态转移方程：

$$dp[i][j] = \begin{cases}
dp[i-1][j-1] + 1 & \text{if } s1[i-1] = s2[j-1] \\
\max(dp[i-1][j], dp[i][j-1]) & \text{otherwise}
\end{cases}$$

4. 图解：

s1 = "abcde"  s2 = "ace"

```
        ""   a    c    e
    ""   0   0    0    0
    a    0   1    1    1
    b    0   1    1    1
    c    0   1    2    2
    d    0   1    2    2
    e    0   1    2    3   ← 得解
```

#### 代码实现

```c
#include <stdio.h>
#include <string.h>

#define MAX_LEN 105

int dp[MAX_LEN][MAX_LEN];

int max(int a, int b) { return a > b ? a : b; }

int lcs(char *s1, char *s2) {
    int m = strlen(s1);
    int n = strlen(s2);

    for (int i = 0; i <= m; i++) dp[i][0] = 0;
    for (int j = 0; j <= n; j++) dp[0][j] = 0;

    for (int i = 1; i <= m; i++) {
        for (int j = 1; j <= n; j++) {
            if (s1[i-1] == s2[j-1])
                dp[i][j] = dp[i-1][j-1] + 1;           // 字符匹配
            else
                dp[i][j] = max(dp[i-1][j], dp[i][j-1]); // 状态转移方程
        }
    }
    return dp[m][n];
}

int main() {
    char s1[] = "abcde";
    char s2[] = "ace";
    printf("LCS 长度: %d\n", lcs(s1, s2));  // 输出: 3
    return 0;
}
```

#### 时间复杂度与优缺点

**时间复杂度** ：$O(mn)$  
**空间复杂度** ：$O(mn)$，可优化为 $O(\min(m,n))$（滚动数组）

**优点** ：
- ✅ 框架适用于序列对齐问题
- ✅ 可扩展到 LCS 具体字符（回溯）

**缺点** ：
- ❌ m、n 都很大时空间压力大

---

### 例 5：区间 DP - 戳气球

#### 问题

戳破气球 i 得分 = `nums[i-1] * nums[i] * nums[i+1]`，求最大总得分。

#### 思路分析

1. **关键思路** ：不想"先戳哪个"，而想"区间 (i,j) 中**最后一个**戳哪个"，这样两侧边界已知，避免状态混乱

2. 定义状态：`dp[i][j]` = 戳破开区间 (i,j) 内所有气球的最大得分

3. 根据分析，我们可以得到这样一个状态转移方程：

$$dp[i][j] = \max_{i < k < j}(dp[i][k] + dp[k][j] + nums[i] \times nums[k] \times nums[j])$$

其中 k 是区间 (i,j) 中最后一个被戳的气球。

#### 代码实现

```c
#include <stdio.h>
#include <string.h>

#define MAX_N 305

int dp[MAX_N][MAX_N];
int nums[MAX_N];

int max(int a, int b) { return a > b ? a : b; }

int maxCoins(int *arr, int n) {
    // 加哨兵边界
    nums[0] = 1;
    for (int i = 1; i <= n; i++) nums[i] = arr[i-1];
    nums[n+1] = 1;
    int N = n + 2;

    memset(dp, 0, sizeof(dp));

    // 按区间长度从小到大枚举
    for (int len = 2; len < N; len++) {
        for (int i = 0; i < N - len; i++) {
            int j = i + len;
            for (int k = i+1; k < j; k++) {   // k 是最后被戳的气球
                int score = dp[i][k] + dp[k][j]
                          + nums[i] * nums[k] * nums[j];
                dp[i][j] = max(dp[i][j], score);  // 状态转移方程
            }
        }
    }
    return dp[0][N-1];
}

int main() {
    int arr[] = {3, 1, 5, 8};
    int n = sizeof(arr) / sizeof(arr[0]);
    printf("最大得分: %d\n", maxCoins(arr, n));  // 输出: 167
    return 0;
}
```

#### 时间复杂度与优缺点

**时间复杂度** ：$O(n^3)$  
**空间复杂度** ：$O(n^2)$

**优点** ：
- ✅ 区间 DP 的经典例题
- ✅ "最后一个"的思路极具启发性
- ✅ 可通过回溯得到具体戳的顺序

**缺点** ：
- ❌ 思路相对复杂，初次接触易困惑
- ❌ 时间复杂度为立方级

---

## 四、DP 模型总结

DP 模型对比总结：

| 类型 | 状态转移方程 | 时间复杂度 | 空间复杂度 | 代表问题 |
|------|------------|----------|---------|---------|
| **线性 DP** | $dp[i] = f(dp[i-1], dp[i-2], \ldots)$ | $O(n)$ | $O(n)$ | 爬楼梯、打家劫舍 |
| **背包 DP** | $dp[i][j] = \max(\text{不放}, \text{放})$ | $O(nW)$ | $O(nW)$ | 0/1 背包、完全背包 |
| **序列 DP** | $dp[i][j] = f(dp[i-1][j-1], ..)$ | $O(mn)$ | $O(mn)$ | LCS、编辑距离 |
| **区间 DP** | $dp[i][j] = \max_k(dp[i][k] + dp[k][j])$ | $O(n^3)$ | $O(n^2)$ | 戳气球、矩阵链乘 |

---

## 总结与建议

1. **从问题出发** ：确定能否用 DP（有最优子结构和重叠子问题）
2. **定义状态** ：清晰地定义 `dp[...]` 的含义
3. **写出转移方程** ：确定状态间的关系
4. **确定初始条件** ：边界情况的处理
5. **实现与优化** ：代码实现，再考虑空间+时间优化
