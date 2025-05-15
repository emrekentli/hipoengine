Merhaba {{ name | upper }}!

{% if isLoggedIn %}
Hoşgeldin tekrar!
{% else %}
Lütfen giriş yapınız.
{% endif %}

Ürünler:
{% for urun in products %}
- {{ urun | title }}
{% endfor %}

Toplam: {{ price | money }} 