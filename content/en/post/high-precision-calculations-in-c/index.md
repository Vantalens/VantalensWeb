---
pinned: false
tags:
    - High-precision calculations
categories:
    - Algorithm
title: "High-precision calculations in C"
description: "Core ideas and implementation methods of high-precision computing in C language"
date: 2026-03-15T22:06:00+08:00
image: ""
math: false
license: ""
hidden: false
comments: true
draft: true
ws_sync_zh_hash: "d6811d24a59bf0ff4b9373fd0269af9583f53ae819136e9f67e7de72cfb45fc2"
---

# C language high precision calculation

The maximum built-in integer `unsigned long long` in C is about $1.8\ times 10 ^ {19} $, which will overflow once the number exceeds this range. The high-precision calculation uses an integer array to store each bit of the large number, and cooperates with the loop simulation vertical operation to break the limit of the number of bits.

## Questions

### Background

Standard shapes are often inadequate in competition and engineering, and typical scenarios include:

- Calculate $100! $ (~ 158 digits)
- Large power operations (e.g. RSA key generation)
- Financial calculations requiring precise results

### Core issues

For two non-negative integers of arbitrary length, add, subtract, multiply, and divide four operations are implemented, and the results are accurate.

### BINDING EFFECT

- Up to $10 ^ 3$ digits (adjustable `MAXN` extension)
- This article only deals with non-negative integers; negative numbers require the addition of symbol bits

## Idea Analysis

### Core Ideas

Save each digit of the large number * * in reverse order * * into the `int' array: `d [0]` for one digit, `d [1]` for ten digits, and so on. The advantage of reverse order is that the carry direction (low → high) is consistent with the growth direction of the array subscript, and the loop is the most natural to write.

### Data Structures

__ code_block_0 __

Layout of the number `12345` in the array:

| Subscript | d [0] | d [1] | d [2] | d [3] | d [4] |
|:----:|:----:|:----:|:----:|:----:|:----:|
| Value | 5 | 4 | 3 | 2 | 1 |

### Key points of each operation

* * Addition * *: add bit by bit, record the carry with the variable `carry`, and loop until the highest carry is also processed.

* * Subtraction * *: Subtract bit by bit, record the borrow with `borrow`, and guarantee $ a\ geq b $ before calling.

* * Multiplication * *: double loop, the result of `a [i] * b [j]` is accumulated to the 'i + j` bit of the result, and finally the carry is processed uniformly. Intermediate results are spill-proof with `long long`.

* * Divide by a small integer * *: starting from the highest position, maintain the remainder `r`, `r = r * 10 + d [i]` at each step, the quotient is` r/b `, and the remainder is updated to` r % b `.

## Code Implementation

### Initialization and I/O

__ code_block_1 __

### Addition and Subtraction

__ code_block_2 __

### Multiplication

__ code_block_3 __

### Divide by small integer

__ code_block_4 __

### Full Demo Program

__ code_block_5 __

### Application: Calculating Factors

__ code_block_6 __

## Complexity and Advantages and Disadvantages

### Time complexity

Set large digits to $ n $:

| Operation | This article implements | Optimization caps |
|:----:|:--------:|:--------:|
| Add/Subtract | $ O (n) $ | — |
| Multiplication | $ O (n ^ 2) $ | $ O (n\ log n) $ (FFT) |
| Divide by small integer | $ O (n) $ | — |
| Factor $ n! $ | $ O (n ^ 2\ cdot\ log n) $ | — |

### Spatial Complexity

$ O (n) $, which is the size of the `d [MAXN]` array in the structure.

### Pros

- The principle is intuitive, fully corresponds to the manual vertical, easy to understand and debug
- Pure C implementation without any external dependencies
- Addition and subtraction $ O (n) $, fully adequate for medium size ($\ leq 10 ^ 4$ bits)

### disadvantages

- Multiplication $ O (n ^ 2) $, slower for very large numbers ($ > 10 ^ 5$ bits)
- Only 1 bit per lattice, the constant factor is too large; it can be changed to 10,000 (4 bits per lattice) to increase the speed by about 4 times
- Negative numbers are not supported for the time being, additional symbolic bit processing needs to be introduced

### Introduction to Perpetual Optimization

Change `base` from 10 to 10000 and store 4 decimal digits per array element:

__ code_block_7 __

The logic of addition, subtraction and multiplication is exactly the same, just change all `% 10` to `% base` and `/10` to `/base`.