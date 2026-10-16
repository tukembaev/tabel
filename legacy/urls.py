from django.conf import settings
from django.conf.urls.static import static
from django.urls import path

from .views import *

    
urlpatterns = [
    path('tabel/', TabelAPIView.as_view()),
    path('mytabel/', MyTabelsAPIView.as_view()),
    path('mytabellast/', MyTabelAPIView.as_view()),
    path('tabel/<int:id>/', NoMyTabelsAPIView.as_view()),
    path('report-tabel/<int:depart_id>/<int:desired_month>/', ReportTabelAPIView.as_view()),
]


urlpatterns += static(settings.STATIC_URL, document_root=settings.STATIC_ROOT)
urlpatterns += static(settings.MEDIA_URL, document_root=settings.MEDIA_ROOT)

#urlpatterns += static(settings.STATIC_URL, document_root=settings.STATIC_ROOT)
