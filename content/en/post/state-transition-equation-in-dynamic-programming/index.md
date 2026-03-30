---
title: "State Transition Equations and Dynamic Programming"
date: 2026-03-02T13:57:00+08:00
description: "Summary of Dynamic Programming and State Transition Equations, and Common Models"
tags:
    - Dynamic programming
    - - State transition equation
categories:
    - Algorithm
draft: false
math: true
comments: true
hidden: false
pinned: false
ws_sync_zh_hash: "885dd1c71fb609cb69d1a0ef22aac03ad0e2ce06429c93e0d44881297fd6691e"
---

In algorithmic problems, we can often see the shadow of dynamic programming, so here is a summary of dynamic programming (DP) and a very important part of it - the state transition equation.

## I. What is Dynamic Planning

Dynamic Programming (DP) is an algorithmic idea that solves the original problem by decomposing it into sub-problems.

Dynamic planning is not some specific data structure, but a way of thinking.

DP needs to meet the following two properties:

### 1. Optimal Substructure

The optimal solution of the original problem contains the optimal solution of the subproblem.

### 2. Overlapping subquestions

Subquestions are computed iteratively and can be cached to avoid duplication.

## II. What is the state transition equation?

To understand the state transition equation, you should first know what a "state" is.

### 1- STATE

Status is a description of the problem at a certain stage, usually denoted by `dp [i]` or `dp [i] [j]`.

For example:
- `dp [i]` = optimal solution for the first i elements
- `dp [i] [j]` = optimal solution from position i to position j
- `dp [i] [w]` = optimal solution considering the first i items with remaining capacity w

### 2. State transition equation

The state transition equation can be roughly written as:

__ code_block_0 __

What the state transition equation does is determine what options are available for that step, as well as the sub-problems behind the choices (which can be understood as recursive).

## III. Typical Examples

### Example 1: Linear DP - climbing stairs

#### Questions

How many ways can you climb 1 or 2 steps at a time to reach step n?

#### Idea Analysis

1. Define status: `dp [i]` = Number of methods to climb to level i
2. Last step analysis: to reach step i, you can only come from step i-1 (step 1) or step i-2 (step 2)
3. Based on the analysis, we can obtain such a state transition equation:

$ $ dp [i] = dp [i-1] + dp [i-2] $ $

Note: There are two more boundary states in this state transition equation: `dp [1] = 1`, `dp [2] = 2`

4. Illustration:

__ code_block_1 __

#### Code Implementation

__ code_block_2 __

#### Time Complexity and Advantages and Disadvantages

* * Time complexity * *: $ O (n) $  
* * Space complexity * *: $ O (n) $

* * Benefits * *:
- The question is simple and easy to understand
- Status definition is intuitive

* * Cons * *:
- Values can only be calculated in one fixed place and need to be recursed multiple times to calculate multiple times

---

### Example 2: Linear DP - Burglary

#### Questions

A row of houses can not rob neighbors, ask for the maximum amount. The given array `nums` represents the amounts for each house.

#### Idea Analysis

1. Define status: `dp [i]` = the maximum amount you can get for grabbing room i
2. Last step analysis: room i, either rob or not rob
   - Grab this room: get `nums [i] + dp [i-2]` (up to i-2 in front)
   - Don't grab this room: get `dp [i-1]` (grab the maximum between i-1)
3. State transition equation:

$ $ dp [i] =\ max (dp [i-1], dp [i-2] + nums [i]) $ $

4. Illustration:

__ code_block_3 __

#### Code Implementation

__ code_block_4 __

#### Time Complexity and Advantages and Disadvantages

* * Time complexity * *: $ O (n) $  
* * Space complexity * *: $ O (n) $, optimized for $ O (1) $ (only the first two are retained)

* * Benefits * *:
- ✅ Similar to climbing stairs, clear thinking
- ✅ Optimizes space to O (1)

* * Cons * *:
- ❌ No direct backtracking on which houses were robbed

---

### Example 3: Backpack DP - 0/1 Backpack

#### Questions

n items, weight `w []`, value `v []`, back capacity W, for maximum value.

#### Idea Analysis

1. Define status: `dp [i] [j]` = maximum value considering the first i items, capacity j
2. Last step analysis: the ith item, put or not put
   - Do not put: `dp [i] [j] = dp [i-1] [j]`
   - Placement: `dp [i] [j] = dp [i-1] [j-w [i]] + v [i]` for $ j\ geq w [i] $
3. State transition equation:

$ $ dp [i] [j] =\ max (dp [i-1] [j], dp [i-1] [j-w [i]] + v [i]) $ $

4. Illustration:

Items: (w = 2, v = 3), (w = 3, v = 4), (w = 4, v = 5) Back carrying capacity W = 5

__ code_block_5 __

#### Code Implementation

__ code_block_6 __

* * Spatial optimization * *: The two-dimensional dp can be compressed into one dimension, and the inner loop * * must be reversed * * to prevent the same item from being repeatedly placed:

__ code_block_7 __

#### Time Complexity and Advantages and Disadvantages

* * Time complexity * *: $ O (nW) $  
* * Space complexity * *: $ O (nW) $, optimized to $ O (W) $

* * Benefits * *:
- DP framework is classic and easy to expand
- Optimal values for all capacities can be solved at once

* * Cons * *:
- When the number or capacity of items is large, the pressure in time and space is high

---

### Example 4: Sequence DP - Longest Common Subsequence (LCS)

#### Questions

Two strings for the length of the longest common subsequence. Example: `"abcde"` and `"ace"` → length is 3 (`ace`)

#### Idea Analysis

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

#### Code Implementation

__ code_block_9 __

#### Time Complexity and Advantages and Disadvantages

* * Time complexity * *: $ O (mn) $  
* * Space complexity * *: $ O (mn) $, optimized for $ O (\ min (m, n)) $ (scrolling array)

* * Benefits * *:
- Frame for sequence alignment problems
- Scalable to LCS specific characters (backtracking)

* * Cons * *:
- When m and n are large, the spatial pressure is large

---

### Example 5: Interval DP - Poke Balloon

#### Questions

Poke the balloon i score = `nums [i-1] * nums [i] * nums [i +1]` to get the maximum total score.

#### Idea Analysis

1. * * Key ideas * *: Do not want to "poke which first", but "poke which last * * in the interval (i, j)", so that the boundaries on both sides are known to avoid confusion

2. Define status: `dp [i] [j]` = Maximum score for all balloons in punctured open interval (i, j)

3. Based on the analysis, we can obtain such a state transition equation:

$ $ dp [i] [j] =\ max_{i < k < j} (dp [i] [k] + dp [k] [j] + nums [i]\ nums [k]\ times nums [j]) $ $

where k is the last poked balloon in interval (i, j).

#### Code Implementation

__ code_block_10 __

#### Time Complexity and Advantages and Disadvantages

* * Time complexity * *: $ O (n ^ 3) $  
* * Space complexity * *: $ O (n ^ 2) $

* * Benefits * *:
- Classic example of an interval DP
- The idea of the "last one" is very enlightening
- The order in which specific stamps can be obtained by backtracking

* * Cons * *:
- The idea is relatively complex and confusing to the first contact
- Time complexity is cubic

---

## IV. DP Model Summary

DP Model Comparison Summary:

| Type | State Transition Equation | Time Complexity | Space Complexity | Represents Problem |
|------|------------|----------|---------|---------|
| * * Linear DP * * | Recursive Relationship | $ O (n) $ | $ O (n) $ | Climbing Stairs, Burglary |
| * * Backpack DP * * | Choose max | $ O (nW) $ | $ O (nW) $ | 0/1 Backpack, full backpack |
| * * Sequence DP * * | Two Pointer Recursion | $ O (mn) $ | $ O (mn) $ | LCS, Edit Distance |
| * * Interval DP * * | Interval Split Recursion | $ O (n ^ 3) $ | $ O (n ^ 2) $ | Balloon Poke, Matrix Chain Multiplication |

---

## Summary & Suggestions

1. * * From the question * *: Determine if DP can be used (with optimal sub-structure and overlapping sub-problems)
2. * * Define status * *: clearly define what `dp [...]` means
3. * * Write out the transfer equation * *: Determine the relationship between the states
4. * * Determination of initial conditions * *: treatment of boundary situations
5. * * Implementation and optimization * *: Code implementation, then consider space + time optimization