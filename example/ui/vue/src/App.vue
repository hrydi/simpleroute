<script setup></script>

<template>
  <nav>
    <router-link to="/">Home</router-link>
    <router-link to="/demo">API Demo</router-link>
  </nav>
  <main>
    <router-view />
  </main>
</template>

<style>
* {
  box-sizing: border-box;
}
body {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
  margin: 0;
  padding: 0;
  background: #fafafa;
  color: #333;
}
#app {
  max-width: 720px;
  margin: 0 auto;
  padding: 24px;
}
nav {
  display: flex;
  gap: 16px;
  margin-bottom: 24px;
  padding-bottom: 12px;
  border-bottom: 1px solid #ddd;
}
nav a {
  color: #333;
  text-decoration: none;
  font-weight: 500;
}
nav a:hover {
  color: #000;
}
nav a.router-link-active {
  color: #1a73e8;
}
</style>
