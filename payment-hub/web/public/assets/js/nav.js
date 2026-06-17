(function () {
  var nav = document.getElementById('site-nav-links')
  if (!nav) return

  fetch('/public/pages')
    .then(function (r) { return r.json() })
    .then(function (data) {
      var pages = data.pages || []
      if (!pages.length) return
      var contact = nav.querySelector('a[href="/contact"]')
      pages.forEach(function (p) {
        var a = document.createElement('a')
        a.href = p.url || ('/' + p.slug)
        a.textContent = p.label || p.slug
        if (contact) nav.insertBefore(a, contact)
        else nav.appendChild(a)
      })
    })
    .catch(function () {})
})()
