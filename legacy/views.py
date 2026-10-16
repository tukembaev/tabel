from rest_framework.permissions import IsAuthenticated
from rest_framework.response import Response
from rest_framework.views import APIView
from rest_framework import generics, status
from config.service import get_employee
import os
from rest_framework.parsers import FileUploadParser, MultiPartParser

from .models import Employee_Enter_Auth
from .serializers import TabelSerializer, ReportTabelSerializer
from datetime import date
import datetime
from apps_logic.calendar.models import Calendar
from apps_logic.employees.models import Employee

from apps_logic.univer.serializers import *
from apps_logic.employees.serializers import *
import requests


class TabelAPIView(generics.ListCreateAPIView):
    parser_classes = (MultiPartParser, FileUploadParser)
    queryset = Employee_Enter_Auth.objects.all()
    serializer_class = TabelSerializer 
    
    def post(self, request, format=None):
        request.data._mutable = True
        data = request.data
        employee = Employee.objects.get(user_id=request.user.id)
        data['employee'] = employee.id
        serializer = TabelSerializer(data=request.data, context={'request': request})
        serializer.is_valid(raise_exception=True)
        infos = Employee_Enter_Auth.objects.filter(employee=employee)
        for i in infos:
            if i.auditorium_end == None:
                pass
            else:
                working_time = i.time_end - i.time
                hours, remainder = divmod(working_time.seconds, 3600)
                minutes, seconds = divmod(remainder, 60)
                formatted_working_time = "{:02}:{:02}:{:02}".format(hours, minutes, seconds)
                i.working_time = formatted_working_time
                i.status_info = "Завершен"
                i.save()
               
        # date_info = date.today()
        # info = Employee_Enter_Auth.objects.get(employee=employee, date_day=date_info)
        # info.geo_end = data['geo']
        # info.auditorium_end = data['auditorium']
        # info.status_info ="Завершен"
        # info.time_end = datetime.datetime.now() 
        # info.image_end = data['image']
        # info.save()
        # date_info = date.today()
      
        # info = Employee_Enter_Auth.objects.get(employee=employee, date_day=date_info)
        # info.geo_end = data['geo']
        # info.auditorium_end = data['auditorium']
        # info.status_info ="Завершен"
        # info.time_end = datetime.datetime.now() 
        # info.image_end = data['image']
        # info.save()
        # meta = []
        # meta["employee"] = employee
        # meta["title"] = "Отметка"
        # meta["description"] = "Ваша отметка от" + str(i.date_day)
        # meta["end"] = i.time_end
        # meta["start"] = i.time
        # meta["color"] = "#43444A"
        # CalendarSerializer(data=meta)
        try:
            date_info = date.today()
            info = Employee_Enter_Auth.objects.get(employee=employee, date_day=date_info)
            info.geo_end = data['geo']
            info.auditorium_end = data['auditorium']
            info.status_info ="Завершен"
            info.time_end = datetime.datetime.now() 
            # info.image_end = data['image']
            info.save()
          
        
            title = "Отметка"
            description = "Ваша отметка от " + str(info.date_day)
            color = "#9ed2ff"
            Calendar.objects.create(
                employee=employee,
                title=title, description=description,
                start= info.time,
                end=info.time_end,
                color=color  
            )
        except:
            date_info = date.today()
            info = Employee_Enter_Auth.objects.create(employee=employee,
                geo = data['geo'],
                auditorium = data['auditorium'],
                status_info ="Начат",
                # image = data['image'],
             )
        # files = request.FILES.get('image')
       
        # files = data['image']
        # if files: 
        #         FileTabel.objects.create(tabel=tabel, file=files, status_info=status_info)

        
        return Response(serializer.data)


# class NoMyTabelsAllAPIView(generics.ListAPIView):
#     queryset = Movement.objects.filter(employee__user__is_active=True, employee__user_type="E").prefetch_related("employee").prefetch_related("divisions").prefetch_related("employee__user")
#     serializer_class = MovPositionUniverNoCriteriaSerializer
#     pagination_class = AllEmployeesStudentsPagination
#     filter_backends = [DjangoFilterBackend, filters.SearchFilter]
#     search_fields = ["employee__first_name", "employee__surname", "employee__last_name"]

#     def list(self, request, *args, **kwargs):
#         response = super().list(request, *args, **kwargs)
#         return response

class NoMyTabelsAPIView(APIView):
    permission_classes = [IsAuthenticated]

    def get(self, request, id):
        # employee = get_employee(id)
        empl = Employee.objects.get(user_id=request.user.id)
        infos = Employee_Enter_Auth.objects.filter(employee=empl)
        for i in infos:
            if i.date_day != date.today():
                if i.status_info != "Завершен":
                    i.status_info = "Не отмечен"
                    i.save()
                    
        for i in infos:
            if i.auditorium_end == None:
                pass
            else:
                working_time = i.time_end - i.time
                hours, remainder = divmod(working_time.seconds, 3600)
                minutes, seconds = divmod(remainder, 60)
                formatted_working_time = "{:02}:{:02}:{:02}".format(hours, minutes, seconds)
                i.working_time = formatted_working_time
                i.status_info = "Завершен"
                i.save()
        info_data = Employee_Enter_Auth.objects.filter(employee=empl)
        serializer = TabelSerializer(info_data, many=True)
        return Response(serializer.data)

# class MyTabelsAPIView(APIView):
#     permission_classes = [IsAuthenticated]

#     def get(self, request):
#         # employee = get_employee(request.user.id)
#         empl = Employee.objects.get(user_id=request.user.id)
#         infos = Employee_Enter_Auth.objects.filter(employee=empl)
#         for i in infos:
#             if i.auditorium_end == None:
#                 pass
#             else:
#                 working_time = i.time_end - i.time
#                 if working_time > datetime.timedelta(hours=8):
#                     formatted_working_time = "08:00:00"
#                 else:
#                     hours, remainder = divmod(working_time.seconds, 3600)
#                     minutes, seconds = divmod(remainder, 60)
#                     formatted_working_time = "{:02}:{:02}:{:02}".format(hours, minutes, seconds)
#                 i.working_time = formatted_working_time
#                 i.status_info = "Завершен"
#                 i.save()
#         serializer = TabelSerializer(infos, many=True)
#         return Response(serializer.data)

class MyTabelsAPIView(APIView):
    permission_classes = [IsAuthenticated]

    def get(self, request):
        # 1. Получаем сотрудника и его записи
        try:
            empl = Employee.objects.get(user_id=request.user.id)
        except Employee.DoesNotExist:
            return Response({"error": "Employee not found"}, status=404)

        infos = Employee_Enter_Auth.objects.filter(employee=empl)

        # 2. Логика расчета рабочего времени (ваша существующая логика)
        for i in infos:
            if i.auditorium_end is not None:
                working_time = i.time_end - i.time
                if working_time > datetime.timedelta(hours=8):
                    formatted_working_time = "08:00:00"
                else:
                    hours, remainder = divmod(working_time.seconds, 3600)
                    minutes, seconds = divmod(remainder, 60)
                    formatted_working_time = "{:02}:{:02}:{:02}".format(hours, minutes, seconds)
                
                i.working_time = formatted_working_time
                i.status_info = "Завершен"
                i.save()

        # 3. Сериализуем список записей
        serializer = TabelSerializer(infos, many=True)
        data = serializer.data # Это список словарей

        # 4. Обогащаем каждую запись данными из внешнего API
        external_api_url = "https://qr.kstu.kg/api/qrcode/"
        
        for item in data:
            # Предполагаем, что поле называется 'auditorium' в вашем сериализаторе
            # Если оно называется иначе (например 'auditorium_end'), замените здесь:
            auditorium_value = item.get('auditorium') 

            if auditorium_value:
                try:
                    # Делаем запрос для конкретной аудитории
                    response = requests.get(external_api_url, params={'end_url': auditorium_value}, timeout=3)
                    
                    if response.status_code == 200:
                        qr_data_list = response.json()
                        if isinstance(qr_data_list, list) and len(qr_data_list) > 0:
                            qr_item = qr_data_list[0]
                            
                            # Добавляем только нужные поля на тот же уровень
                            item['cumpus'] = qr_item.get('cumpus')
                            item['korpus'] = qr_item.get('korpus')
                            item['name2'] = qr_item.get('name2')
                            
                except requests.exceptions.RequestException:
                    # Если сервер QR недоступен, просто оставляем поля пустыми
                    item['qr_error'] = "service_unavailable"

        return Response(data)

# class MyTabelAPIView(APIView):
#     permission_classes = [IsAuthenticated]

#     def get(self, request):
#         empl = Employee.objects.get(user_id=request.user.id)
#         date_info = date.today()
#         # info = Employee_Enter_Auth.objects.get(employee=employee, date_day=date_info)
#         try:
#             # date_info = date.today()
#             info = Employee_Enter_Auth.objects.get(employee=empl, date_day=date_info)
#         except:
#             info = None
#         serializer = TabelSerializer(info)
#         return Response(serializer.data)


class MyTabelAPIView(APIView):
    permission_classes = [IsAuthenticated]

    def get(self, request):
        # 1. Получаем сотрудника
        try:
            empl = Employee.objects.get(user_id=request.user.id)
        except Employee.DoesNotExist:
            return Response({"error": "Сотрудник не найден"}, status=status.HTTP_404_NOT_FOUND)

        # 2. Получаем запись посещаемости за сегодня
        date_info = date.today()
        try:
            info = Employee_Enter_Auth.objects.get(employee=empl, date_day=date_info)
            serializer = TabelSerializer(info)
            data = serializer.data 
        except Employee_Enter_Auth.DoesNotExist:
            return Response({"status": "Запись за сегодня не найдена"}, status=200)

        # 3. HTTP запрос к внешнему серверу QR
        if info.auditorium:
            external_api_url = "https://qr.kstu.kg/api/qrcode/"
            params = {'end_url': info.auditorium}
            
            try:
                response = requests.get(external_api_url, params=params, timeout=5)
                
                if response.status_code == 200:
                    qr_data_list = response.json()
                    
                    if isinstance(qr_data_list, list) and len(qr_data_list) > 0:
                        # Берем первый объект из списка
                        qr_item = qr_data_list[0]
                        
                        # Выбираем только те поля, которые вам нужны
                        # Используем .get(), чтобы код не упал, если какого-то поля нет
                        data['cumpus'] = qr_item.get('cumpus')
                        data['korpus'] = qr_item.get('korpus')
                        data['name2'] = qr_item.get('name2')
                        
            except requests.exceptions.RequestException:
                # В случае ошибки сети просто оставляем поля пустыми или пишем ошибку
                data['qr_status'] = "error_fetching_external_data"

        return Response(data)
        

class ReportTabelAPIView(APIView):
    permission_classes = [IsAuthenticated]

    def get(self, request, depart_id, desired_month):
        empls = Employee.objects.filter(movement__divisions_id=depart_id).prefetch_related('movement__divisions')
        serializer = ReportTabelSerializer(empls, many=True, context={'desired_month': desired_month})
        return Response(serializer.data)