<template>
  <div class="app-logo">
    <!-- 
      Logo 优化版：
      1. 背景：使用主题色径向渐变 (primary-300 -> primary-600)，解决"太白"和"太深"的矛盾。
      2. 线条：使用白色半透明，确保在深色背景上的高对比度和科技感。
    -->
    <!-- 
      Logo 三线波浪版 (Flux Lines):
      1. 核心概念：回归经典的三条数据流线。
      2. 视觉元素：使用渐变色线条描绘数据流动，无背景圆，简洁明快。
      3. 适配性：透明背景，静态展示。
    -->
    <svg viewBox="0 0 160 160" xmlns="http://www.w3.org/2000/svg" class="w-full h-full">
      <defs>
        <linearGradient id="line-grad" x1="0%" y1="0%" x2="100%" y2="0%">
          <stop offset="0%" style="stop-color:rgb(var(--primary-300-color));stop-opacity:1" />
          <stop offset="100%" style="stop-color:rgb(var(--primary-600-color));stop-opacity:1" />
        </linearGradient>
      </defs>
      
      <!-- 三线波浪 (Three Flux Lines): 静态 -->
      <g fill="none" stroke="url(#line-grad)" stroke-width="8" stroke-linecap="round" stroke-linejoin="round">
        <!-- 上层波浪 -->
        <path d="M 20 50 Q 50 20 80 50 T 140 50" stroke-opacity="0.4" />
        <!-- 中层波浪 -->
        <path d="M 20 80 Q 50 50 80 80 T 140 80" stroke-opacity="0.7" />
        <!-- 下层波浪 -->
        <path d="M 20 110 Q 50 80 80 110 T 140 110" stroke-opacity="1.0" />
      </g>
      
      <!-- 核心点 -->
      <circle cx="80" cy="80" r="8" fill="#FDE047" stroke="none" />
    </svg>
  </div>
</template>

<style scoped>
</style>
