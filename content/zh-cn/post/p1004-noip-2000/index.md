---
tags:
    - 动态规划
    - DFS
    - 费用流
categories:
    - 算法
pinned: false
title: "P1004 [NOIP 2000 提高组] 方格取数 分析与总结"
description: "经典棋盘模型问题的多种解法分析：动态规划、DFS 记忆化和费用流"
date: 2026-02-28T11:31:00+08:00
image: ""
math: true
license: ""
hidden: false
comments: true
draft: false
---

这是一道经典但具有一定难度的棋盘模型题。虽然是 2000 年的 NOIP 题目，但作为压轴题，对于第一次接触的人来说还是相当困难的。本文总结了三种不同的解法：动态规划、DFS 记忆化和最小费用最大流，从易到难逐步展开。

## 问题

### 题目来源
NOIP 2000 提高组 T4

### 问题描述

设有 N×N 的方格图 (N≤9)，我们将其中的某些方格中填入正整数，而其他的方格中则放入数字 0。
某人从图的左上角的 A 点（0，0）出发，可以向下行走，也可以向右走，直到到达右下角的 B 点（N, N)。在走过的路上，他可以取走方格中的数（取走后的方格中将变为数字 0）。
此人从 A 点到 B 点共走两次，试找出 2 条这样的路径，使得取得的数之和为最大。

### 输入输出

**输入格式** ：输入的第一行为一个整数 N（表示 N×N 的方格图），接下来的每行有三个整数，前两个表示位置，第三个数为该位置上所放的数。一行单独的 0 表示输入结束。

**输出格式** ：只需输出一个整数，表示 2 条路径上取得的最大的和。

### 样例

输入：
```
8
2 3 13
2 6  6
3 5  7
4 4 14
5 2 21
5 6  4
6 3 15
7 2 14
0 0  0
```

输出：
```
67
```

### 约束条件
- 数据范围：1≤N≤9

### 问题分析

为什么不能采取两次枚举（即先找一条最优路径，再在剩余格子中找第二条）？因为一次路径会改变地图（取走数字），影响第二次的结果，所以两条路径必须联动考虑，而不能独立优化。

## 思路分析

三种解法的思路分别如下：

### 解法一：动态规划

**核心思想** ：将两条路径同步推进，用一个 DP 同时模拟两个人的行走。

**状态设计** ：设 `dp[k][x1][x2]`，其中：
- k 为当前走的步数（即 x + y 的值，从 2 到 2N）
- x1、x2 分别为两人当前所在的行号
- 由 k 和 x 可以推出 y = k - x（关键优化，这一步降低了维度），因此列号不需要单独存储

**去重处理** ：当两人在同一格时（x1 == x2，则 y1 == y2），该格只取一次。

**状态转移** ：每步两人各自可以选择向右或向下，共 4 种组合。每个状态 dp[k][x1][x2] 代表走到第 k 步，人 1 在行 x1、人 2 在行 x2 时的最大取数和。

### 解法二：DFS + 记忆化搜索

**核心思想** ：类似于 DP 算法（毕竟深搜和 DP 本质上就是一个东西），但添加了记忆化搜索来避免重复计算。若无记忆化搜索，计算量将是指数级爆炸，在 N=9 时约为 $4^{16}$ 次。

**实现方式** ：从初始状态出发，递归地尝试所有可能的转移，同时用 memo 数组缓存已计算过的状态，避免重复。在达到终点时返回 0，逐层返回最大值。

### 解法三：费用流（最小费用最大流）

**核心思想** ：将"两条路径取最大值"转化为网络流问题。

**建模思路** ：
- 两条从 A 到 B 的路径 = 从源点到汇点流量为 2 的流
- 每个格子最多取一次 = 每个节点容量限制
- 取得数字最大 = 费用最大（转为最小费用取负值）

**拆点处理** ：每个格子 (i,j) 拆成两个节点 in 和 out：
- in → out 容量 1，费用 -map[i][j]（第一条路径取数）
- in → out 再加容量 1，费用 0（第二条路径经过但不取数）

**连边** ：(i,j) 的 out 连向 (i+1,j) 的 in 和 (i,j+1) 的 in，容量 2，费用 0。

## 代码实现

### 解法一：动态规划

完整代码
```c
#include <stdio.h>

int N;
int map[10][10];
int dp[20][10][10]; // dp[步数][人1的行][人2的行]

int main() {
    scanf("%d", &N);

    int x, y, v;
    while (scanf("%d %d %d", &x, &y, &v) && (x || y || v))
        map[x][y] = v;

    // 初始化为-1表示不可达
    for (int k = 0; k < 20; k++)
        for (int i = 0; i < 10; i++)
            for (int j = 0; j < 10; j++)
                dp[k][i][j] = -1;

    dp[2][1][1] = map[1][1];

    // ========== 动态规划主循环 ==========
    for (int k = 2; k < 2 * N; k++) {
        for (int x1 = 1; x1 <= N; x1++) {
            int y1 = k - x1;
            if (y1 < 1 || y1 > N) continue;

            for (int x2 = x1; x2 <= N; x2++) {
                int y2 = k - x2;
                if (y2 < 1 || y2 > N) continue;
                if (dp[k][x1][x2] < 0) continue;

                // 尝试所有4种移动组合
                for (int m1 = 0; m1 <= 1; m1++) {
                    for (int m2 = 0; m2 <= 1; m2++) {
                        int nx1 = x1 + m1,     ny1 = y1 + (1 - m1);
                        int nx2 = x2 + m2,     ny2 = y2 + (1 - m2);

                        if (nx1 > N || ny1 > N) continue;
                        if (nx2 > N || ny2 > N) continue;

                        // 计算当前步收益
                        int gain = map[nx1][ny1];
                        if (nx1 != nx2) gain += map[nx2][ny2];

                        // 规范化：保证 a <= b
                        int a = nx1, b = nx2;
                        if (a > b) { int t = a; a = b; b = t; }

                        int newval = dp[k][x1][x2] + gain;
                        if (newval > dp[k + 1][a][b])
                            dp[k + 1][a][b] = newval;
                    }
                }
            }
        }
    }

    printf("%d\n", dp[2 * N][N][N]);
    return 0;
}
```

### 解法二：DFS + 记忆化搜索

```c
#include <stdio.h>
#include <string.h>

// ========== DFS + 记忆化搜索 ==========

int N;
int map[10][10];
int memo[20][10][10];    // 记忆化数组
int visited[20][10][10]; // 标记是否计算过

int dfs(int k, int x1, int x2) {
    if (x1 == N && x2 == N) return 0;

    if (visited[k][x1][x2]) return memo[k][x1][x2];
    visited[k][x1][x2] = 1;

    int y1 = k - x1;
    int y2 = k - x2;
    int best = -1;

    for (int m1 = 0; m1 <= 1; m1++) {
        for (int m2 = 0; m2 <= 1; m2++) {
            int nx1 = x1 + m1,  ny1 = y1 + (1 - m1);
            int nx2 = x2 + m2,  ny2 = y2 + (1 - m2);

            if (nx1 > N || ny1 > N) continue;
            if (nx2 > N || ny2 > N) continue;

            int a = nx1, b = nx2;
            if (a > b) { int t = a; a = b; b = t; }

            int sub = dfs(k + 1, a, b);
            if (sub < 0) continue;

            int gain = map[nx1][ny1];
            if (nx1 != nx2) gain += map[nx2][ny2];

            if (gain + sub > best) best = gain + sub;
        }
    }

    memo[k][x1][x2] = best;
    return best;
}

int main() {
    scanf("%d", &N);

    int x, y, v;
    while (scanf("%d %d %d", &x, &y, &v) && (x || y || v))
        map[x][y] = v;

    memset(visited, 0, sizeof(visited));

    int start_val = map[1][1];
    int result = dfs(2, 1, 1);

    printf("%d\n", result >= 0 ? start_val + result : 0);
    return 0;
}
```

### 解法三：费用流（最小费用最大流）

```c
#include <stdio.h>
#include <string.h>

#define MAXN 1000
#define MAXE 10000
#define INF  0x3f3f3f3f

// ========== 链式前向星数据结构 ==========

int head[MAXN], nxt[MAXE], to[MAXE], cap[MAXE], cost[MAXE], tot;

void init() {
    memset(head, -1, sizeof(head));
    tot = 0;
}

void add_edge(int u, int v, int c, int w) {
    to[tot] = v; cap[tot] = c; cost[tot] = w; nxt[tot] = head[u]; head[u] = tot++;
    to[tot] = u; cap[tot] = 0; cost[tot] = -w; nxt[tot] = head[v]; head[v] = tot++;
}

int dist[MAXN], in_queue[MAXN], prevv[MAXN], preve[MAXN];
int queue[MAXN * 100];

// ========== SPFA算法 ==========

int spfa(int s, int t, int n) {
    memset(dist, 0x3f, sizeof(int) * (n + 1));
    memset(in_queue, 0, sizeof(int) * (n + 1));
    dist[s] = 0;

    int front = 0, rear = 0;
    queue[rear++] = s;
    in_queue[s] = 1;

    // SPFA主循环
    while (front != rear) {
        int u = queue[front++];
        in_queue[u] = 0;

        // 松弛边
        for (int e = head[u]; e != -1; e = nxt[e]) {
            if (cap[e] > 0 && dist[to[e]] > dist[u] + cost[e]) {
                dist[to[e]] = dist[u] + cost[e];
                prevv[to[e]] = u;
                preve[to[e]] = e;

                if (!in_queue[to[e]]) {
                    queue[rear++] = to[e];
                    in_queue[to[e]] = 1;
                }
            }
        }
    }

    return dist[t] < INF;
}

// ========== 最小费用最大流 ==========

int mcmf(int s, int t, int n) {
    int total_cost = 0;

    while (spfa(s, t, n)) {
        int flow = INF;
        for (int v = t; v != s; v = prevv[v])
            if (cap[preve[v]] < flow) flow = cap[preve[v]];

        for (int v = t; v != s; v = prevv[v]) {
            cap[preve[v]] -= flow;
            cap[preve[v] ^ 1] += flow;
        }

        total_cost += dist[t] * flow;
    }

    return total_cost;
}

int main() {
    int N;
    scanf("%d", &N);

    int map[10][10] = {0};
    int x, y, v;
    while (scanf("%d %d %d", &x, &y, &v) && (x || y || v))
        map[x][y] = v;

    init();

    int S = 2 * N * N + 1;
    int T = 2 * N * N + 2;
    int total_nodes = T;

    #define IN(i,j)  ((i-1)*N+(j))
    #define OUT(i,j) (N*N+(i-1)*N+(j))

    for (int i = 1; i <= N; i++) {
        for (int j = 1; j <= N; j++) {
            if (map[i][j] > 0) {
                add_edge(IN(i,j), OUT(i,j), 1, -map[i][j]);
                add_edge(IN(i,j), OUT(i,j), 1, 0);
            } else {
                add_edge(IN(i,j), OUT(i,j), 2, 0);
            }

            if (j + 1 <= N) add_edge(OUT(i,j), IN(i,j+1), 2, 0);
            if (i + 1 <= N) add_edge(OUT(i,j), IN(i+1,j), 2, 0);
        }
    }

    add_edge(S, IN(1,1), 2, 0);
    add_edge(OUT(N,N), T, 2, 0);

    int ans = -mcmf(S, T, total_nodes);
    printf("%d\n", ans);
    return 0;
}
```

## 时间复杂度与优缺点

### 解法一：动态规划

**时间复杂度** ：$O(N^3)$  
- 状态数：$O(N^2) \times O(N^2) / 2 = O(N^4)$，但由于降维和 x1、x2 的约束，实际为 $O(N^3)$
- 每个状态有 4 次转移

**空间复杂度** ：$O(N^3)$  
- dp 数组大小为 $2N \times N \times N$

**优点** ：
- ✅ 思路清晰，易于理解
- ✅ 一次遍历解决问题
- ✅ 代码相对简洁

**缺点** ：
- ❌ 空间占用较大（N=9 时约 1.3MB）

### 解法二：DFS + 记忆化搜索

**时间复杂度** ：$O(N^3)$  
- 状态数与 DP 相同
- 每个状态最多计算一次（记忆化）

**空间复杂度** ：$O(N^3)$  
- memo 和 visited 数组各占 $O(N^3)$

**优点** ：
- ✅ 逻辑自然，从上向下思考
- ✅ 易于添加剪枝（虽然此题中剪枝不多）
- ✅ 可灵活调整状态定义

**缺点** ：
- ❌ 递归调用栈深度为 $O(N)$（栈空间）
- ❌ 空间复杂度与 DP 相同

### 解法三：费用流

**时间复杂度** ：$O(Flow \times SPFA) = O(2 \times E \log V) = O(N^2 \times N^2) = O(N^4)$  
- 流量为 2
- SPFA 每次 $O(V \log V)$，其中 $V = O(N^2)$，$E = O(N^2)$

**空间复杂度** ：$O(V + E) = O(N^2)$  
- 图的存储空间

**优点** ：
- ✅ 适用于更一般的场景（多条路径、带限制的图等）
- ✅ 代码框架可复用于其他费用流问题
- ✅ 空间占用相对较少

**缺点** ：
- ❌ 时间复杂度最高（约 $O(N^4)$ vs $O(N^3)$）
- ❌ 代码长且复杂，易出错
- ❌ 难度陡升，超出竞赛难度范围

### 对比与总结

| 特性 | 解法一 DP | 解法二 DFS | 解法三 费用流 |
|------|----------|----------|----------|
| 易理解度 | ★★★★☆ | ★★★★☆ | ★☆☆☆☆ |
| 实现难度 | ★★☆☆☆ | ★★☆☆☆ | ★★★★★ |
| 时间复杂度 | $O(N^3)$ | $O(N^3)$ | $O(N^4)$ |
| 空间复杂度 | $O(N^3)$ | $O(N^3)$ | $O(N^2)$ |
| 推荐指数 | ★★★★★ | ★★★★☆ | ★★☆☆☆ |

**结论** ：对于此题，**解法一（DP）** 是最佳选择，既清晰高效又不过度复杂。解法二适合想要练习 DFS 的同学。解法三虽然优雅，但对于 N≤9 的规模效率不如 DP，仅作扩展知识。


