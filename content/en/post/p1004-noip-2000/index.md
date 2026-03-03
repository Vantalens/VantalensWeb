---
tags:
    - 动态规划
    - DFS
    - 费用流
categories:
    - 算法
pinned: false
title: "P1004 [NOIP 2000 improvement group] grid score analysis and summary"
description: "经典棋盘模型问题的多种解法分析：动态规划、DFS 记忆化和费用流"
date: 2026-02-28T11:31:00+08:00
image: ""
math: true
license: ""
hidden: false
comments: true
draft: false
ws_sync_zh_hash: "8e2d9fabdeb44c4bc5a279e90a64f0ec4c8c65516e934f8ee4a5f0e7828f4a13"
---

This is a classic but difficult chessboard problem. Although it is the topic of NOIP in 2000, it is still quite difficult for first-time contacts as the final topic. This paper summarizes three different solutions: Dynamic Planning, DFS Memorization, and Minimum Cost Maximum Flow, which unfolds gradually from easy to difficult.

## 问题

### 题目来源
NOIP 2000 提高组 T4

### 问题描述

There is a square diagram of N × N (N ≤ 9), some of which we fill in positive integers, while others put the number 0.
Someone starts at point A (0, 0) in the top left corner of the diagram and can walk down or to the right until they reach point B (N, N) in the bottom right corner.On the way he walks, he can take the number in the square (which becomes the number 0).
This person walked from point A to point B twice, trying to find 2 such paths, so that the sum of the obtained numbers is the maximum.

### 输入输出

* * Input format * *: The first line of input is an integer N (representing the grid diagram of N × N), and each subsequent line has three integers, the first two represent the position, and the third number is the number placed on the position. A separate line of 0 indicates the end of the input.

* * Output format * *: Just output an integer representing the maximum sum obtained on the 2 paths.

### 样例

Input:
__ code_block_0 __

Output:
__ code_block_1 __

### 约束条件
- 数据范围：1≤N≤9

### 问题分析

Why can't I enumerate twice (i.e. find an optimal path first, then find the second one in the remaining cells)? Because one path changes the map (takes the number) and affects the result of the second, the two paths must be considered in conjunction and cannot be optimized independently.

## 思路分析

The ideas for the three solutions are as follows:

### 解法一：动态规划

* * Core idea * *: Advance both paths simultaneously, simulating two people walking at the same time with one DP.

* * State design * *: set `dp [k] [x1] [x2]`, where:
- k is the number of steps currently taken (i.e. the value of x + y, from 2 to 2N)
- x1 and x2 are the line numbers where the two people are currently located
- y = k - x can be derived from k and x (key optimization, this step reduces the dimension), so column numbers do not need to be stored separately

* * Deduplication * *: When two people are in the same cell (x1 = = x2, y1 = = y2), the cell is taken only once.

* * State transition * *: Two people each can choose to move to the right or down for a total of 4 combinations. Each state dp [k] [x1] [x2] represents the sum of the maximum values at line x1 and line x2 for human 1 and 2, respectively, going to step k.

### 解法二：DFS + 记忆化搜索

* * Core idea * *: Similar to the DP algorithm (deep search and DP are essentially one thing after all), but adds memorized search to avoid double counting. If there is no memorized search, the amount of calculation will be an exponential explosion, about $4 ^ {16} $ times at N = 9.

* * Implementation * *: Starting from the initial state, all possible transitions are attempted recursively, while the calculated state is cached with the memo array, avoiding duplication. Returns 0 when the end point is reached and returns the maximum value layer by layer.

### 解法三：费用流（最小费用最大流）

* * Core Idea * *: Convert "maximize two paths" into a network flow problem.

* * Modeling ideas * *:
- Two paths from A to B = 2 flows from the source to the sink
- Fetch up to once per cell = Capacity limit per node
- Get number max = cost max (min to min)

* * Split point processing * *: Each grid (i, j) is split into two nodes in and out:
- in → out capacity 1, cost - map [i] [j] (first path taken)
- in → out plus capacity 1, cost 0 (second path through but not counted)

* * Connecting edges * *: (i, j) out connects (i +1, j) in and (i, j +1) in, capacity 2, cost 0.

## 代码实现

### 解法一：动态规划

Full Code
__ code_block_2 __

### 解法二：DFS + 记忆化搜索

__ code_block_3 __

### 解法三：费用流（最小费用最大流）

__ code_block_4 __

## 时间复杂度与优缺点

### 解法一：动态规划

* * Time complexity * *: $ O (N ^ 3) $  
- Number of states: $ O (N ^ 2)\ times O (N ^ 2)/2 = O (N ^ 4) $, but due to the constraints of dimensionality reduction and x1 and x2, it is actually $ O (N ^ 3) $
- 4 transitions per state

* * Space complexity * *: $ O (N ^ 3) $  
- dp array size is $2N\ times N\ times N $

* * Benefits * *:
- Clear ✅ thinking and easy to understand
- Problem solving in ✅ one iteration
- ✅ The code is relatively simple

* * Cons * *:
- Large ❌ footprint (approx. 1.3MB at N = 9)

### 解法二：DFS + 记忆化搜索

* * Time complexity * *: $ O (N ^ 3) $  
- Same number of statuses as DP
- Up to one calculation per state (memorization)

* * Space complexity * *: $ O (N ^ 3) $  
- Memo and visited arrays account for $ O (N ^ 3) $ each

* * Benefits * *:
- ✅ Logical and natural, thinking from top to bottom
- ✅ Easy to add pruning (although there are not many pruning in this question)
- ✅ Flexible state definition

* * Cons * *:
- The ❌ recursive call stack depth is $ O (N) $ (stack space)
- Same ❌ space complexity as DP

### 解法三：费用流

* * Time complexity * *: $ O (Flow\ times SPFA) = O (2\ times E\ log V) = O (N ^ 2\ times N ^ 2) = O (N ^ 4) $  
- Flow is 2
- SPFA $ O (V\ log V) $ each time, where $ V = O (N ^ 2) $, $ E = O (N ^ 2) $

* * Space complexity * *: $ O (V + E) = O (N ^ 2) $  
- Storage space for diagrams

* * Benefits * *:
- ✅ Suitable for more general scenarios (multiple paths, restricted diagrams, etc.)
- ✅ Code frameworks can be reused for other expense stream issues
- Relatively sparsely occupied ✅ space

* * Cons * *:
- Highest ❌ time complexity (about $ O (N ^ 4) $ vs $ O (N ^ 3) $)
- ❌ The code is long, complex and error-prone
- ❌ Difficulty has skyrocketed beyond the limits of the competition

### 对比与总结

| Features | Solution 1 DP | Solution 2 DFS | Solution 3 Cost Flow |
|------|----------|----------|----------|
| Ease of Understanding | ★★★★☆ | ★★★★☆ | ★☆☆☆☆ |
| Difficulty | ★★☆☆☆ | ★★☆☆☆ | ★★★★★ |
| Time Complexity | $ O (N ^ 3) $ | $ O (N ^ 3) $ | $ O (N ^ 4) $ |
| Space Complexity | $ O (N ^ 3) $ | $ O (N ^ 3) $ | $ O (N ^ 2) $ |
| Recommendation Index | ★★★★★ | ★★★★☆ | ★★☆☆☆ |

* * Conclusion * *: For this question, * * Solution 1 (DP) * * is the best choice, which is clear, efficient and not overly complicated. Solution 2 is for students who want to practice DFS. Solution 3 Although elegant, it is not as efficient as DP for the scale of N ≤ 9, and only extends knowledge.


