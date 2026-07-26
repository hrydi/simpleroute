<script setup></script>

<template>
  <h1>SimpleRoute Example</h1>
  <p>
    This demo app showcases the <strong>simpleroute</strong> Go HTTP router.
    The API endpoints are registered in <code>example/user.go</code>.
  </p>
  <h2>Available Routes</h2>
  <table>
    <thead>
      <tr><th>Method</th><th>Path</th><th>Description</th></tr>
    </thead>
    <tbody>
      <tr><td>GET</td><td><code>/user</code></td><td>Plain-text greeting</td></tr>
      <tr><td>GET</td><td><code>/user/profile</code></td><td>JSON user profile</td></tr>
      <tr><td>GET</td><td><code>/user/{id}</code></td><td>Path parameter demo</td></tr>
    </tbody>
  </table>
</template>

<style scoped>
table {
  border-collapse: collapse;
  width: 100%;
  max-width: 500px;
}
th, td {
  border: 1px solid #ccc;
  padding: 8px 12px;
  text-align: left;
}
th {
  background: #f5f5f5;
}
code {
  background: #eee;
  padding: 2px 6px;
  border-radius: 3px;
}
</style>
