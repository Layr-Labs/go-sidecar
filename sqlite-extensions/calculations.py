from decimal import Decimal, getcontext, ROUND_HALF_UP, ROUND_UP, ROUND_DOWN, Context, ROUND_HALF_EVEN
import math

def preNileTokensPerDay(tokens: str) -> str:
    big_amount = float(tokens)
    div = 0.999999999999999
    res = big_amount * div

    res_str = "{}".format(res)
    return "{}".format(int(Decimal(res_str)))

def amazonStakerTokenRewards(sp:str, tpd:str) -> str:
    # Set precision to 38 to match DECIMAL(38,0)
    getcontext().prec = 15

    # Convert string inputs to Decimal, preserving original precision
    proportion = Decimal(sp)
    tokens = Decimal(tpd)

    # Perform the multiplication
    result = proportion * tokens

    # Convert to string, ensuring no scientific notation
    return "{}".format(int(result))

def nileStakerTokenRewards(sp:str, tpd:str) -> str:
    getcontext().prec = 16
    getcontext().rounding = ROUND_UP
    # Convert string inputs to Decimal, preserving original precision
    proportion = Decimal(sp)
    tokens = Decimal(tpd)

    # Perform the multiplication
    result = proportion * tokens

    getcontext().prec = 38
    res_decimal = result.quantize(Decimal('1'), rounding=ROUND_UP)

    # Convert to string, ensuring no scientific notation
    return "{}".format(res_decimal, 'f')

def stakerTokenRewards(sp:str, tpd:str) -> str:
    getcontext().prec = 38
    getcontext().rounding = ROUND_HALF_EVEN
    stakerProportion = Decimal(sp)
    tokensPerDay = Decimal(tpd)

    decimal_res = stakerProportion * tokensPerDay

    floored = decimal_res.quantize(Decimal('1'), rounding=ROUND_DOWN)
    return "{}".format(floored, 'f')


def amazonOperatorTokenRewards(totalStakerOperatorTokens:str) -> str:
    getcontext().prec = 38
    totalStakerOperatorTokens = Decimal(totalStakerOperatorTokens)

    operatorTokens = totalStakerOperatorTokens * Decimal(0.1)

    rounded = operatorTokens.quantize(Decimal('1'), rounding=ROUND_HALF_UP)

    return "{}".format(rounded)

def nileOperatorTokenRewards(totalStakerOperatorTokens:str) -> str:
    if totalStakerOperatorTokens[-1] == "0":
        return "{}".format(int(totalStakerOperatorTokens) // 10)
    totalStakerOperatorTokens = Decimal(totalStakerOperatorTokens)
    operatorTokens = Decimal(str(totalStakerOperatorTokens)) * Decimal(0.1)
    rounded = operatorTokens.quantize(Decimal('1'), rounding=ROUND_HALF_UP)
    return "{}".format(rounded)

def bigGt(a:str, b:str) -> bool:
    return Decimal(a) > Decimal(b)

def sumBigC(a:str, b:str) -> str:
    sum = Decimal(a) + Decimal(b)
    return format(sum, 'f')

def numericMultiplyC(a:str, b:str) -> str:
    product = Decimal(a) * Decimal(b)

    return format(product, 'f')

def calculateStakerProportion(stakerWeightStr: str, totalWeightStr: str) -> str:
    getcontext().prec = 15
    getcontext().rounding = ROUND_DOWN

    stakerWeight = Decimal(stakerWeightStr)
    totalWeight = Decimal(totalWeightStr)

    preProportion = stakerWeight / totalWeight

    preProportion1 = math.floor(float(preProportion * Decimal('1000000000000000')))

    preProportion2 = preProportion1 / Decimal('1000000000000000')

    return "{}".format(preProportion2, 'f')

#print("Actual  ", stakerTokenRewards('0.07148054555830600000', '3.9285714285714246e+17'))
#print("expected 28081642897905928")
#print('\n')
#print("Actual  ", nileStakerTokenRewards('0.013636363636363', '3571428571428568000000000000000000000'))
#print("expected 48701298701296390000000000000000000")
#exit()
#print('\n')
#print("Actual  ", nileStakerTokenRewards('', ''))
#print("expected ")
#print('\n')
#print("Actual  ", nileStakerTokenRewards('', ''))
#print("expected ")
