(function () {
  function formatPrice(n) {
    return '₹' + Number(n).toLocaleString('en-IN')
  }

  function formatLimit(n) {
    return Number(n).toLocaleString('en-IN') + ' QR requests'
  }

  function parseFeatures(json) {
    if (!json) return []
    try {
      var list = JSON.parse(json)
      return Array.isArray(list) ? list : []
    } catch (e) {
      return []
    }
  }

  function renderPlan(p) {
    var card = document.createElement('div')
    card.className = 'plan-card' + (p.is_recommended ? ' recommended' : '')
    var features = parseFeatures(p.features_json)
    var featHtml = features.map(function (f) {
      var ok = f.included !== false
      return '<li><span class="' + (ok ? 'yes' : 'no') + '"><i class="fas fa-' + (ok ? 'check' : 'times') + '"></i></span> ' + (f.text || '') + '</li>'
    }).join('')
    if (!featHtml) {
      featHtml = '<li><span class="yes"><i class="fas fa-check"></i></span> 0% transaction fee</li>' +
        '<li><span class="yes"><i class="fas fa-check"></i></span> Webhook callbacks</li>'
    }
    card.innerHTML =
      (p.is_recommended ? '<span class="plan-badge">Recommended</span>' : '') +
      '<div class="plan-name">' + p.name + '</div>' +
      '<div class="plan-price">' + formatPrice(p.price_inr) + ' <small>/ ' + p.validity_days + ' days</small></div>' +
      '<div class="plan-limit">' + formatLimit(p.order_limit) + '</div>' +
      '<ul class="plan-features">' + featHtml + '</ul>' +
      '<a href="/dashboard/register" class="btn ' + (p.is_recommended ? 'btn-primary' : 'btn-outline') + '">Buy Now ' + formatPrice(p.price_inr) + '</a>'
    return card
  }

  var grid = document.getElementById('pricing-grid')
  if (!grid) return

  fetch('/public/plans')
    .then(function (r) { return r.json() })
    .then(function (data) {
      grid.innerHTML = ''
      ;(data.plans || []).forEach(function (p) {
        grid.appendChild(renderPlan(p))
      })
    })
    .catch(function () {
      grid.innerHTML = '<p class="muted">Unable to load plans. Please refresh or contact support.</p>'
    })
})()
