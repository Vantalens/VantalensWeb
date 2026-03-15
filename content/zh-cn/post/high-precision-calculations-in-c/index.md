---
pinned: false
tags:
    - 高精度计算
categories:
    - 算法
title: "C 语言的高精度计算"
description: "关于 C 语言中高精度计算的核心思想与实现方式"
date: 2026-03-15T22:06:00+08:00
image: ""
math: false
license: ""
hidden: false
comments: true
draft: true
---

# C 语言高精度计算

C 语言最大内置整型 `unsigned long long` 约为 $1.8 \times 10^{19}$，一旦数字超出这个范围就会溢出。高精度计算用整数数组存储大数的每一位，配合循环模拟竖式运算，从而突破位数限制。

## 问题

### 背景

标准整型在竞赛和工程中常常不够用，典型场景包括：

- 计算 $100!$（约 158 位）
- 大数幂运算（如 RSA 密钥生成）
- 需要精确结果的金融计算

### 核心问题

对两个任意长度的非负整数，实现加、减、乘、除四则运算，结果精确无误差。

### 约束条件

- 数字位数最多 $10^3$ 位（可调整 `MAXN` 扩展）
- 本文只处理非负整数；负数需额外引入符号位

## 思路分析

### 核心思想

把大数的每一位**逆序**存入 `int` 数组：`d[0]` 存个位，`d[1]` 存十位，以此类推。逆序的好处是进位方向（低位 → 高位）与数组下标增长方向一致，循环写起来最自然。

### 数据结构

```c
#define MAXN 1000

typedef struct {
    int d[MAXN];    /* d[0] 存最低位（逆序） */
    int len;        /* 当前有效位数 */
} BigInt;
```

数字 `12345` 在数组中的布局：

| 下标 | d[0] | d[1] | d[2] | d[3] | d[4] |
|:----:|:----:|:----:|:----:|:----:|:----:|
| 值   |  5   |  4   |  3   |  2   |  1   |

### 各运算的关键点

**加法**：逐位相加，用变量 `carry` 记录进位，循环直到最高位进位也处理完。

**减法**：逐位相减，用 `borrow` 记录借位，要求调用前保证 $a \geq b$。

**乘法**：双重循环，`a[i] * b[j]` 的结果累加到结果的第 `i+j` 位，最后统一处理进位。中间结果用 `long long` 防溢出。

**除以小整数**：从最高位开始，维护余数 `r`，每步 `r = r * 10 + d[i]`，商位为 `r / b`，余数更新为 `r % b`。

## 代码实现

### 初始化与输入输出

```c
#include <stdio.h>
#include <string.h>

#define MAXN 1000

typedef struct {
    int d[MAXN];
    int len;
} BigInt;

void init(BigInt *a) {
    // 清零所有位，防止遗留垃圾值影响计算
    memset(a->d, 0, sizeof(a->d));
    a->len = 1;
}

/* 从字符串读入，自动完成逆序存储 */
void fromStr(BigInt *a, const char *s) {
    init(a);
    int n = strlen(s);
    a->len = n;
    for (int i = 0; i < n; i++)
        a->d[i] = s[n - 1 - i] - '0';   /* 逆序：末位 → d[0] */
}

/* 打印：从最高位到最低位输出 */
void print(const BigInt *a) {
    for (int i = a->len - 1; i >= 0; i--)
        printf("%d", a->d[i]);
    printf("\n");
}

/* 比较大小，返回 1(a>b) / 0(a==b) / -1(a<b) */
int cmp(const BigInt *a, const BigInt *b) {
    if (a->len != b->len)
        return a->len > b->len ? 1 : -1;
    for (int i = a->len - 1; i >= 0; i--)
        if (a->d[i] != b->d[i])
            return a->d[i] > b->d[i] ? 1 : -1;
    return 0;
}
```

### 加法与减法

```c
/* c = a + b */
void add(const BigInt *a, const BigInt *b, BigInt *c) {
    init(c);
    int carry = 0;
    int n = a->len > b->len ? a->len : b->len;

    for (int i = 0; i < n || carry; i++) {
        int sum = carry;
        if (i < a->len) sum += a->d[i];
        if (i < b->len) sum += b->d[i];
        c->d[i] = sum % 10;
        carry   = sum / 10;
        c->len  = i + 1;
    }
}

/* c = a - b，要求 a >= b */
void sub(const BigInt *a, const BigInt *b, BigInt *c) {
    init(c);
    int borrow = 0;

    for (int i = 0; i < a->len; i++) {
        int diff = a->d[i] - borrow - (i < b->len ? b->d[i] : 0);
        if (diff < 0) { diff += 10; borrow = 1; }
        else          borrow = 0;
        c->d[i] = diff;
    }

    c->len = a->len;
    // 去除前导零，但至少保留 1 位
    while (c->len > 1 && c->d[c->len - 1] == 0) c->len--;
}
```

### 乘法

```c
/* c = a * b */
void mul(const BigInt *a, const BigInt *b, BigInt *c) {
    init(c);
    c->len = a->len + b->len;   /* 结果最多 len_a + len_b 位 */

    for (int i = 0; i < a->len; i++) {
        int carry = 0;
        for (int j = 0; j < b->len || carry; j++) {
            // 用 long long 防止 int 相乘溢出
            long long cur = (long long)c->d[i + j]
                          + (long long)a->d[i] * (j < b->len ? b->d[j] : 0)
                          + carry;
            c->d[i + j] = cur % 10;
            carry        = cur / 10;
        }
    }

    while (c->len > 1 && c->d[c->len - 1] == 0) c->len--;
}
```

### 除以小整数

```c
/* 商存入 q，余数通过 rem 传出；b 为普通 int */
void divSmall(const BigInt *a, int b, BigInt *q, int *rem) {
    init(q);
    q->len = a->len;
    long long r = 0;

    // 从最高位开始模拟长除法
    for (int i = a->len - 1; i >= 0; i--) {
        r = r * 10 + a->d[i];
        q->d[i] = (int)(r / b);
        r       = r % b;
    }

    *rem = (int)r;
    while (q->len > 1 && q->d[q->len - 1] == 0) q->len--;
}
```

### 完整演示程序

```c
int main(void) {
    BigInt a, b, c;

    // ========== 加法演示 ==========
    fromStr(&a, "99999999999999999999");   /* 20 位 9 */
    fromStr(&b, "1");
    add(&a, &b, &c);
    printf("加法: "); print(&c);           /* 输出 10^20 */

    // ========== 乘法演示 ==========
    fromStr(&a, "123456789");
    fromStr(&b, "987654321");
    mul(&a, &b, &c);
    printf("乘法: "); print(&c);           /* 输出 121932631112635269 */

    // ========== 除法演示 ==========
    int rem;
    BigInt q;
    fromStr(&a, "123456789");
    divSmall(&a, 7, &q, &rem);
    printf("除法商: "); print(&q);
    printf("余数: %d\n", rem);

    return 0;
}
```

### 应用：计算阶乘

```c
/* 计算 n! 并打印 */
void factorial(int n) {
    BigInt result, tmp;
    char buf[20];

    fromStr(&result, "1");
    for (int i = 2; i <= n; i++) {
        sprintf(buf, "%d", i);
        fromStr(&tmp, buf);
        mul(&result, &tmp, &result);
    }

    printf("%d! = ", n); print(&result);
}
```

## 复杂度与优缺点

### 时间复杂度

设大数位数为 $n$：

| 运算 | 本文实现 | 优化上限 |
|:----:|:--------:|:--------:|
| 加 / 减 | $O(n)$ | — |
| 乘法 | $O(n^2)$ | $O(n \log n)$（FFT） |
| 除以小整数 | $O(n)$ | — |
| 阶乘 $n!$ | $O(n^2 \cdot \log n)$ | — |

### 空间复杂度

$O(n)$，即结构体中的 `d[MAXN]` 数组大小。

### 优点

-  原理直观，与手工竖式完全对应，易于理解和调试
-  纯 C 实现，无任何外部依赖
-  加减法 $O(n)$，对中等规模（$\leq 10^4$ 位）完全够用

### 缺点

-  乘法 $O(n^2)$，对超大数（$> 10^5$ 位）较慢
-  每格只存 1 位，常数因子偏大；可改为万进制（每格存 4 位）提速约 4 倍
-  暂不支持负数，需额外引入符号位处理

### 万进制优化简介

将 `BASE` 从 10 改为 10000，每个数组元素存 4 位十进制数：

```c
#define BASE  10000
#define BASEW 4      /* 每格宽度，用于打印补零 */

/* 打印时：最高格不补零，其余各格补足 4 位 */
printf("%d", a->d[a->len - 1]);
for (int i = a->len - 2; i >= 0; i--)
    printf("%04d", a->d[i]);
```

加减乘的逻辑完全相同，只需把所有 `% 10` 改为 `% BASE`，`/ 10` 改为 `/ BASE`。