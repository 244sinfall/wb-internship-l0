
document.getElementById('search-form').addEventListener('submit', (e) => {
    e.preventDefault()
    const orderId = document.getElementById('order-input').value
    const searchDb = document.getElementById('db-search').checked
    if(orderId) {
        let path = `../orders/${orderId}`
        if(searchDb) path+='?db'
        window.location.href = path
    }
})

for(let item of document.querySelectorAll(".unix-time")) {
    item.innerHTML = new Date(Number(item.innerHTML * 1000)).toLocaleDateString("en-US")
}