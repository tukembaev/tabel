from rest_framework import serializers

from .models import *
from apps_logic.employees.models import Employee
from datetime import datetime, timedelta
from django.db.models.functions import ExtractMonth, ExtractYear  
import logging 
import numpy as np
import calendar

class TabelSerializer(serializers.ModelSerializer):
    employee_name = serializers.CharField(source="employee.surname_name", read_only=True)
    user_id = serializers.CharField(source="employee.user.id", read_only=True)
    # image = serializers.URLField(source="creator.imeag.url", read_only=True)
    class Meta:
        model = Employee_Enter_Auth
        fields = ('id', 'time', 'date_day','image','image_end', 'employee', 'employee_name', 'user_id',  'auditorium', 'auditorium_end', 'time_end','geo', 'geo_end' , 'working_time', 'status_info')
    
# locale.setlocale(locale.LC_TIME, 'ru_RU.UTF-8')

class ReportTabelSerializer(serializers.ModelSerializer):
    employee_name = serializers.CharField(source="surname_name", read_only=True)
    subdivision = serializers.CharField(source="movement.categories", read_only=True)
    rate = serializers.FloatField(source="movement.stavka", read_only=True)
    position_name = serializers.CharField(source="movement.position", read_only=True)
    division_name = serializers.CharField(source="movement.divisions.division_name", read_only=True)
    tabels = serializers.SerializerMethodField()

    class Meta:
        model = Employee
        fields = ('id', 'employee_name', 'subdivision', "position_name", "division_name", 'rate', 'tabels')

    def str_to_timedelta(self, time_str):
        h, m, s = map(int, time_str.split(':'))
        return timedelta(hours=h, minutes=m, seconds=s)
    
    def timedelta_to_str(self, td):
        total_seconds = int(td.total_seconds())
        hours = total_seconds // 3600
        minutes = (total_seconds % 3600) // 60
        seconds = total_seconds % 60
        return f"{hours:02}:{minutes:02}:{seconds:02}"
    
    def count_working_days(self, year, month):
        first_day = datetime(year, month, 1)
        next_month = month % 12 + 1
        last_day = datetime(year if next_month > month else year + 1, next_month, 1) - timedelta(days=1)
        days_in_month = np.arange(first_day, last_day + timedelta(days=1), timedelta(days=1)).astype(datetime)
        working_days = np.sum(np.isin([d.weekday() for d in days_in_month], [0, 1, 2, 3, 4]))
        return working_days

    def get_tabels(self, obj):
        current_year = datetime.now().year
        # 1. Получаем месяц из контекста корректно
        desired_month = self.context.get('desired_month')

        # 2. Определяем первый и последний день месяца правильно
        first_day_of_month = datetime(current_year, desired_month, 1)
        
        # calendar.monthrange возвращает кортеж (день_недели_первого_дня, количество_дней_в_месяце)
        _, last_day = calendar.monthrange(current_year, desired_month)
        last_day_of_month = datetime(current_year, desired_month, last_day)
        
        russian_weekdays = {
            'Monday': 'Понедельник', 'Tuesday': 'Вторник','Wednesday': 'Среда',
            'Thursday': 'Четверг', 'Friday': 'Пятница', 'Saturday': 'Суббота', 'Sunday': 'Воскресенье'
        }

        # Генерация всех дней
        all_days_in_month = [first_day_of_month + timedelta(days=i) for i in range(last_day)]
        
        all_days_dict = {
            day.strftime('%d.%m.%Y'): {
                'date_day': day.strftime('%d.%m.%Y'), 
                'day_of_week': russian_weekdays[day.strftime('%A')], 
                'working_time': "Выходной" if day.weekday() >= 5  else "Нет отметки"
            } for day in all_days_in_month
        }

        # Фильтрация данных из БД
        employees = Employee_Enter_Auth.objects.filter(
            employee=obj
        ).annotate(
            month=ExtractMonth('time'), 
            year=ExtractYear('time')
        ).filter(
            month=desired_month, 
            year=current_year
        ).prefetch_related('employee')
        
        tabels_data = TabelSerializer(employees, many=True).data

        # Обновляем словарь данными из БД
        for entry in tabels_data:
            date_day = entry['date_day']
            if date_day in all_days_dict:
                # Если в этот день есть отметка, заменяем "Нет отметки" на реальные данные
                all_days_dict[date_day] = entry

        # Превращаем словарь обратно в список
        final_tabels = list(all_days_dict.values())
        
        # Логика статусов "День не наступил" и подсчет времени
        now = datetime.now()
        total_duration = timedelta()

        for item in final_tabels:
            # Превращаем строку даты обратно в объект для сравнения
            date_obj = datetime.strptime(item['date_day'], '%d.%m.%Y')
            
            if date_obj.date() > now.date():
                if item['working_time'] == "Нет отметки":
                    item['working_time'] = "День не наступил"
            elif item['working_time'] not in [None, "Выходной", "Нет отметки", "День не наступил"]:
                total_duration += self.str_to_timedelta(item['working_time'])
            
            if item.get('working_time') is None:
                item['working_time'] = "Нет отметки"

        total_time_str = self.timedelta_to_str(total_duration)
        final_tabels.append({"working_count": total_time_str})
        final_tabels.append({"working_days": self.count_working_days(current_year, desired_month)})
        
        return final_tabels
        
    # def get_tabels(self, obj):
    #     current_year = datetime.now().year
    #     desired_month = 11 if self.context.get('desired_month') == 12 else self.context.get('desired_month')
    #     first_day_of_month = datetime(current_year, desired_month, 1)
    #     last_day_of_month = datetime(current_year, desired_month + 1, 1) - timedelta(days=1)
        
    #     russian_weekdays = {
    #         'Monday': 'Понедельник', 'Tuesday': 'Вторник','Wednesday': 'Среда',
    #         'Thursday': 'Четверг', 'Friday': 'Пятница', 'Saturday': 'Суббота', 'Sunday': 'Воскресенье'
    #     }

    #     all_days_in_month = [first_day_of_month + timedelta(days=i) for i in range((last_day_of_month - first_day_of_month).days + 1)]
        
        
    #     all_days_dict = {
    #         day.strftime('%d.%m.%Y'): {
    #             'date_day': day.strftime('%d.%m.%Y'), 
    #             'day_of_week': russian_weekdays[day.strftime('%A')], 
    #             'working_time': "Выходной" if day.weekday() >= 5  else "Нет отметки"
    #         } for day in all_days_in_month
    #     }

    #     employees = Employee_Enter_Auth.objects.filter(employee=obj).annotate(month=ExtractMonth('time'), year=ExtractYear('time')).filter(month=desired_month, year=current_year).prefetch_related('employee')
    #     tabels = TabelSerializer(employees, many=True).data

    #     for obj in tabels:
    #         date_day = obj['date_day']
    #         if date_day in all_days_dict and all_days_dict[date_day]['working_time'] == "Нет отметки":
    #             obj['day_of_week'] = all_days_dict[date_day]['day_of_week']
    #             all_days_dict[date_day] = obj

    #     tabels = list(all_days_dict.values())
    #     today_str = datetime.now().strftime('%d.%m.%Y')
    #     total_duration = timedelta()

    #     for obj in tabels:
    #         date_day = datetime.strptime(obj['date_day'], '%d.%m.%Y')
    #         if date_day < datetime.now():
    #             if obj['working_time'] == "Нет отметки":
    #                 obj['working_time'] = "Нет отметки"
    #         elif date_day > datetime.now():
    #             if obj['working_time'] == "Нет отметки":
    #                 obj['working_time'] = "День не наступил"
    #         elif obj['working_time'] not in [None, "Выходной", "Нет отметки"]:
    #             total_duration += self.str_to_timedelta(obj['working_time'])
    #         if obj['working_time'] is None:
    #             obj['working_time'] = "Нет отметки" # если не отметились в конце дня

    #     total_time_str = self.timedelta_to_str(total_duration)

    #     tabels.append({"working_count": total_time_str})
    #     tabels.append({"working_days": self.count_working_days(current_year, desired_month)})
        
    #     return tabels