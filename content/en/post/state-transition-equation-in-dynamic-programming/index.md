---
title: "State Transition Equations and Dynamic Programming"
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
ws_sync_zh_hash: "d38b34f10b8f70f4b3a79526440c6e91496baf719c5c32efd5de19dd592f8034"
---

In algorithmic problems, we can often see the shadow of dynamic programming, so here is a summary of dynamic programming (DP) and a very important part of it - the state transition equation.

## 一、何为动态规划

Dynamic Programming (DP) is an algorithmic idea that solves the original problem by decomposing it into sub-problems.

Dynamic planning is not some specific data structure, but a way of thinking.

DP needs to meet the following two properties:

### 1. 最优子结构

The optimal solution of the original problem contains the optimal solution of the subproblem.

### 2. 重叠子问题

Subquestions are computed iteratively and can be cached to avoid duplication.

## 二、何为状态转移方程

To understand the state transition equation, you should first know what a "state" is.

### 1. "状态"

Status is a description of the problem at a certain stage, usually denoted by `dp [i]` or `dp [i] [j]`.

For example:
- `dp [i]` = optimal solution for the first i elements
- `dp [i] [j]` = optimal solution from position i to position j
- `dp [i] [w]` = optimal solution considering the first i items with remaining capacity w

### 2. 状态转移方程

The state transition equation can be roughly written as:

__ code_block_0 __

What the state transition equation does is determine what options are available for that step, as well as the sub-problems behind the choices (which can be understood as recursive).

## 三、典型例题

## 三、典型例题

### 例 1：线性 DP - 爬楼梯

#### 问题

How many ways can you climb 1 or 2 steps at a time to reach step n?

#### 思路分析

1. Define status: `dp [i]` = Number of methods to climb to level i
2. Last step analysis: to reach step i, you can only come from step i-1 (step 1) or step i-2 (step 2)
3. Based on the analysis, we can obtain such a state transition equation:

$ $ dp [i] = dp [i-1] + dp [i-2] $ $

Note: There are two more boundary states in this state transition equation: `dp [1] = 1`, `dp [2] = 2`

4. Illustration:

__ code_block_1 __

#### 代码实现

__ code_block_2 __

#### 时间复杂度与优缺点

* * Time complexity * *: $ O (n) $  
* * Space complexity * *: $ O (n) $

* * Benefits * *:
- ✅ The question is simple and easy to understand
- ✅ Status definition is intuitive

* * Cons * *:
- Values ❌ can only be calculated in one fixed place and need to be recursed multiple times to calculate multiple times

---

### 例 2：线性 DP - 打家劫舍

#### 问题

A row of houses can not rob neighbors, ask for the maximum amount. The given array `nums` represents the amounts for each house.

#### 思路分析

1. Define status: `dp [i]` = the maximum amount you can get for grabbing room i
2. Last step analysis: room i, either rob or not rob
   - Grab this room: get `nums [i] + dp [i-2]` (up to i-2 in front)
   - Don't grab this room: get `dp [i-1]` (grab the maximum between i-1)
3. State transition equation:

$ $ dp [i] =\ max (dp [i-1], dp [i-2] + nums [i]) $ $

4. Illustration:

__ code_block_3 __

#### 代码实现

__ code_block_4 __

#### 时间复杂度与优缺点

* * Time complexity * *: $ O (n) $  
* * Space complexity * *: $ O (n) $, optimized for $ O (1) $ (only the first two are retained)

* * Benefits * *:
- ✅ Similar to climbing stairs, clear thinking
- ✅ Optimizes space to O (1)

* * Cons * *:
- ❌ No direct backtracking on which houses were robbed

---

### 例 3：背包 DP - 0/1 背包

#### 问题

n items, weight `w []`, value `v []`, back capacity W, for maximum value.

#### 思路分析

1. Define status: `dp [i] [j]` = maximum value considering the first i items, capacity j
2. Last step analysis: the ith item, put or not put
   - Do not put: `dp [i] [j] = dp [i-1] [j]`
   - Placement: `dp [i] [j] = dp [i-1] [j-w [i]] + v [i]` for $ j\ geq w [i] $
3. State transition equation:

$ $ dp [i] [j] =\ max (dp [i-1] [j], dp [i-1] [j-w [i]] + v [i]) $ $

4. Illustration:

Items: (w = 2, v = 3), (w = 3, v = 4), (w = 4, v = 5) Back carrying capacity W = 5

__ code_block_5 __

#### 代码实现

__ code_block_6 __

* * Spatial optimization * *: The two-dimensional dp can be compressed into one dimension, and the inner loop * * must be reversed * * to prevent the same item from being repeatedly placed:

__ code_block_7 __

#### 时间复杂度与优缺点

* * Time complexity * *: $ O (nW) $  
* * Space complexity * *: $ O (nW) $, optimized to $ O (W) $

* * Benefits * *:
- ✅ DP framework is classic and easy to expand
- Optimal values for all capacities ✅ can be solved at once

* * Cons * *:
- When the number or capacity of ❌ items is large, the pressure in time and space is high

---

### 例 4：序列 DP - 最长公共子序列（LCS）

#### 问题

Two strings for the length of the longest common subsequence. Example: `"abcde"` and `"ace"` → length is 3 (`ace`)

#### 思路分析

1. Define the status: `dp [i] [j]` = LCS length of the first i characters of s1 and the first j characters of s2
2. Last step analysis: Are s1 [i] and s2 [j] equal:
   - Equals: `dp [i] [j] = dp [i-1] [j-1] + 1`
   - unequal: `dp [i] [j] = max (dp [i-1] [j], dp [i] [j-1])` (remove s1 [i] or s2 [j] to see which is better)

3. State transition equation: 'dp [i] [j] = dp [i-1] [j-1] + 1` when s1 [i-1] = = s2 [j-1]; otherwise `dp [i] [j] = max (dp [i-1] [j], dp [i] [j-1])`

Or as a piecewise function:

$ $ dp [i] [j] =\ begin {cases}
\ text {dp [i-1] [j-1] + 1} &\ text {when equal}\\
\ text {max (dp [i-1] [j], dp [i] [j-1])} &\ text {when not equal}
\ end {cases} $ $

4. Illustration:

s1 = "abcde" s2 = "ace"

__ code_block_8 __

#### 代码实现

__ code_block_9 __

#### 时间复杂度与优缺点

* * Time complexity * *: $ O (mn) $  
* * Space complexity * *: $ O (mn) $, optimized for $ O (\ min (m, n)) $ (scrolling array)

* * Benefits * *:
- ✅ Frame for sequence alignment problems
- ✅ Scalable to LCS specific characters (backtracking)

* * Cons * *:
- When ❌ m and n are large, the spatial pressure is large

---

### 例 5：区间 DP - 戳气球

#### 问题

Poke the balloon i score = `nums [i-1] * nums [i] * nums [i +1]` to get the maximum total score.

#### 思路分析

1. * * Key ideas * *: Do not want to "poke which first", but "poke which last * * in the interval (i, j)", so that the boundaries on both sides are known to avoid confusion

2. Define status: `dp [i] [j]` = Maximum score for all balloons in punctured open interval (i, j)

3. Based on the analysis, we can obtain such a state transition equation:

$ $ dp [i] [j] =\ max_{i < k < j} (dp [i] [k] + dp [k] [j] + nums [i]\ nums [k]\ times nums [j]) $ $

where k is the last poked balloon in interval (i, j).

#### 代码实现

__ code_block_10 __

#### 时间复杂度与优缺点

* * Time complexity * *: $ O (n ^ 3) $  
* * Space complexity * *: $ O (n ^ 2) $

* * Benefits * *:
- Classic example of an ✅ interval DP
- The idea of the ✅ "last one" is very enlightening
- ✅ The order in which specific stamps can be obtained by backtracking

* * Cons * *:
- The ❌ idea is relatively complex and confusing to the first contact
- ❌ Time complexity is cubic

---

## 四、DP 模型总结

DP Model Comparison Summary:

| Type | State Transition Equation | Time Complexity | Space Complexity | Represents Problem |
|------|------------|----------|---------|---------|
| * * Linear DP * * | Recursive Relationship | $ O (n) $ | $ O (n) $ | Climbing Stairs, Burglary |
| * * Backpack DP * * | Choose max | $ O (nW) $ | $ O (nW) $ | 0/1 Backpack, full backpack |
| * * Sequence DP * * | Two Pointer Recursion | $ O (mn) $ | $ O (mn) $ | LCS, Edit Distance |
| * * Interval DP * * | Interval Split Recursion | $ O (n ^ 3) $ | $ O (n ^ 2) $ | Balloon Poke, Matrix Chain Multiplication |

---

## 总结与建议

1. * * From the question * *: Determine if DP can be used (with optimal sub-structure and overlapping sub-problems)
2. * * Define status * *: clearly define what `dp [...]` means
3. * * Write out the transfer equation * *: Determine the relationship between the states
4. * * Determination of initial conditions * *: treatment of boundary situations
5. * * Implementation and optimization * *: Code implementation, then consider space + time optimization